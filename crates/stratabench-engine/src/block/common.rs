use rand::Rng;
use std::fs::{File, OpenOptions};
use std::os::unix::fs::OpenOptionsExt;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use crate::config::{
    parse_block_bytes, parse_size_bytes, EngineConfig, EngineResults, IntervalSample, LatencyUS,
};
use crate::progress;

pub const MAX_LATENCY_SAMPLES: usize = 100_000;

pub struct RunState {
    pub bs: usize,
    pub duration: Duration,
    pub start: Instant,
    pub total_ops: Arc<AtomicU64>,
    pub read_ops: Arc<AtomicU64>,
    pub write_ops: Arc<AtomicU64>,
    pub bucket_ops: Arc<Vec<AtomicU64>>,
    pub latencies: Arc<Mutex<Vec<f64>>>,
}

impl RunState {
    pub fn new(cfg: &EngineConfig, bs: usize) -> Self {
        let buckets = cfg.duration_sec.max(1) as usize;
        Self {
            bs,
            duration: Duration::from_secs(cfg.duration_sec.max(1) as u64),
            start: Instant::now(),
            total_ops: Arc::new(AtomicU64::new(0)),
            read_ops: Arc::new(AtomicU64::new(0)),
            write_ops: Arc::new(AtomicU64::new(0)),
            bucket_ops: Arc::new((0..buckets).map(|_| AtomicU64::new(0)).collect()),
            latencies: Arc::new(Mutex::new(Vec::with_capacity(4096))),
        }
    }

    pub fn record(&self, write: bool, latency_us: f64) {
        self.total_ops.fetch_add(1, Ordering::Relaxed);
        if write {
            self.write_ops.fetch_add(1, Ordering::Relaxed);
        } else {
            self.read_ops.fetch_add(1, Ordering::Relaxed);
        }
        let sec = self.start.elapsed().as_secs() as usize;
        if sec < self.bucket_ops.len() {
            self.bucket_ops[sec].fetch_add(1, Ordering::Relaxed);
        }
        if let Ok(mut samples) = self.latencies.lock() {
            if samples.len() < MAX_LATENCY_SAMPLES {
                samples.push(latency_us);
            } else {
                let idx = self.total_ops.load(Ordering::Relaxed) as usize % MAX_LATENCY_SAMPLES;
                samples[idx] = latency_us;
            }
        }
    }

    pub fn finish(&self) -> EngineResults {
        let elapsed = self.start.elapsed().as_secs_f64().max(0.001);
        let ops = self.total_ops.load(Ordering::Relaxed) as f64;
        let reads = self.read_ops.load(Ordering::Relaxed) as f64;
        let writes = self.write_ops.load(Ordering::Relaxed) as f64;
        let iops = ops / elapsed;
        let mbps = ops * self.bs as f64 / (1024.0 * 1024.0) / elapsed;
        let mut intervals = Vec::new();
        for (i, b) in self.bucket_ops.iter().enumerate() {
            let n = b.load(Ordering::Relaxed) as f64;
            if n <= 0.0 {
                continue;
            }
            intervals.push(IntervalSample {
                seq: (i + 1) as i32,
                elapsed_sec: 1.0,
                iops: n,
                throughput_mbps: n * self.bs as f64 / (1024.0 * 1024.0),
                avg_latency_us: None,
            });
        }
        EngineResults {
            iops,
            read_iops: Some(reads / elapsed),
            write_iops: Some(writes / elapsed),
            throughput_mbps: mbps,
            ops_per_sec: iops,
            total_operations: Some(ops as i64),
            latency_us: percentile_latency(&self.latencies, iops),
            intervals,
        }
    }
}

pub fn validate_block_cfg(cfg: &EngineConfig) -> Result<(usize, u64), String> {
    if cfg.target.is_empty() {
        return Err("empty target".into());
    }
    if cfg.layer != "block" {
        return Err("not a block layer profile".into());
    }
    let bs = align_up(parse_block_bytes(&cfg.block_size) as usize, 4096);
    let dataset = align_down(parse_size_bytes(&cfg.dataset_size), bs);
    if dataset < bs as u64 {
        return Err("dataset smaller than block size".into());
    }
    Ok((bs, dataset))
}

pub fn open_target(path: &str, direct: bool) -> Result<File, String> {
    if direct {
        match OpenOptions::new()
            .read(true)
            .write(true)
            .custom_flags(libc::O_DIRECT)
            .open(path)
        {
            Ok(f) => return Ok(f),
            Err(e) => {
                eprintln!("stratabench-engine: O_DIRECT open failed ({e}), using buffered I/O")
            }
        }
    }
    OpenOptions::new()
        .read(true)
        .write(true)
        .open(path)
        .map_err(|e| format!("open {path}: {e}"))
}

pub fn spawn_progress(
    cfg: &EngineConfig,
    bucket_ops: Arc<Vec<AtomicU64>>,
    bs: usize,
    start: Instant,
    duration: Duration,
) -> Option<std::thread::JoinHandle<()>> {
    progress::spawn_progress_writer(cfg.progress_path.clone(), bucket_ops, bs, start, duration)
}

pub fn should_write(pattern: &str, mix: i32, rng: &mut impl Rng) -> bool {
    let p = pattern.to_lowercase();
    if p.contains("write") && !p.contains("read") {
        return true;
    }
    if p.contains("read") && !p.contains("write") && !p.contains("rw") {
        return false;
    }
    let read_pct = if mix > 0 { mix } else { 70 };
    rng.gen_range(0..100) >= read_pct
}

pub fn pick_offset(
    pattern: &str,
    dataset: u64,
    bs: usize,
    seq_off: &mut u64,
    tid: usize,
    rng: &mut impl Rng,
) -> u64 {
    let max_off = dataset.saturating_sub(bs as u64);
    if max_off == 0 {
        return 0;
    }
    let p = pattern.to_lowercase();
    if p.contains("rand") {
        let steps = max_off / bs as u64;
        let step = rng.gen_range(0..=steps);
        step * bs as u64
    } else {
        let off = *seq_off % max_off;
        *seq_off = seq_off.wrapping_add(bs as u64 * (tid as u64 + 1));
        off
    }
}

pub fn aligned_buffer(size: usize, align: usize) -> Vec<u8> {
    let mut v = vec![0u8; size + align];
    let ptr = v.as_ptr() as usize;
    let pad = (align - ptr % align) % align;
    v.drain(0..pad);
    v.truncate(size);
    v
}

fn percentile_latency(latencies: &Arc<Mutex<Vec<f64>>>, iops: f64) -> LatencyUS {
    let mut v = latencies.lock().unwrap().clone();
    if v.is_empty() {
        let est = if iops > 0.0 { 1_000_000.0 / iops } else { 100.0 };
        return LatencyUS {
            min: Some(est * 0.5),
            mean: Some(est),
            p50: est,
            p95: est * 2.5,
            p99: est * 4.0,
            max: Some(est * 8.0),
        };
    }
    v.sort_by(|a, b| a.partial_cmp(b).unwrap_or(std::cmp::Ordering::Equal));
    let pct = |p: f64| -> f64 {
        let idx = ((v.len() as f64 - 1.0) * p).round() as usize;
        v[idx.min(v.len() - 1)]
    };
    LatencyUS {
        min: Some(*v.first().unwrap()),
        mean: Some(v.iter().sum::<f64>() / v.len() as f64),
        p50: pct(0.50),
        p95: pct(0.95),
        p99: pct(0.99),
        max: Some(*v.last().unwrap()),
    }
}

fn align_up(n: usize, align: usize) -> usize {
    ((n + align - 1) / align) * align
}

fn align_down(n: u64, align: usize) -> u64 {
    (n / align as u64) * align as u64
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn aligned_buffer_len() {
        let b = aligned_buffer(4096, 4096);
        assert_eq!(b.len(), 4096);
        assert_eq!(b.as_ptr() as usize % 4096, 0);
    }

    #[test]
    fn validate_block_cfg_aligns_dataset() {
        let cfg = crate::config::EngineConfig {
            target: "/dev/nvme0n1".into(),
            layer: "block".into(),
            block_size: "4k".into(),
            dataset_size: "1MB".into(),
            duration_sec: 60,
            ramp_sec: 0,
            queue_depth: 32,
            threads: 4,
            read_write_mix: 0,
            direct_io: false,
            ..Default::default()
        };
        let (bs, dataset) = validate_block_cfg(&cfg).unwrap();
        assert_eq!(bs, 4096);
        assert!(dataset >= bs as u64);
    }
}
