use crate::config::{EngineConfig, EngineResults, IntervalSample, LatencyUS};
use std::thread;
use std::time::Duration;

pub fn synthesize_with_progress(cfg: &EngineConfig) -> EngineResults {
    if !cfg.progress_path.is_empty() {
        emit_synthetic_progress(cfg);
    }
    synthesize(cfg)
}

pub fn synthesize(cfg: &EngineConfig) -> EngineResults {
    let threads = cfg.threads.max(1) as f64;
    let qd = cfg.queue_depth.max(1) as f64;
    let duration = cfg.duration_sec.max(1) as f64;
    let mut base = 40_000.0 * threads * qd.sqrt();
    if cfg.layer == "object" {
        base = 4_000.0 * threads;
    }
    if cfg.pattern.to_lowercase().contains("seq") {
        base *= 1.8;
    }
    let read = if cfg.read_write_mix > 0 {
        base * cfg.read_write_mix as f64 / 100.0
    } else {
        base * 0.7
    };
    let write = base - read;
    let p50 = 80.0 + qd * 2.0;
    EngineResults {
        iops: base,
        read_iops: Some(read),
        write_iops: Some(write),
        throughput_mbps: base * crate::config::parse_block_bytes(&cfg.block_size) as f64 / (1024.0 * 1024.0),
        ops_per_sec: base,
        total_operations: Some((base * duration) as i64),
        latency_us: LatencyUS {
            min: Some(p50 * 0.4),
            mean: Some(p50 * 1.1),
            p50,
            p95: p50 * 2.5,
            p99: p50 * 4.0,
            max: Some(p50 * 12.0),
        },
        intervals: Vec::new(),
    }
}

fn emit_synthetic_progress(cfg: &EngineConfig) {
    let preview = synthesize(cfg);
    let mut duration = cfg.duration_sec.max(1);
    if duration > 3 {
        duration = 3;
    }
    let buckets = duration.max(3);
    let tick = Duration::from_millis((duration as u64 * 1000 / buckets as u64).max(100));
    let _ = std::fs::write(&cfg.progress_path, "");
    for i in 1..=buckets {
        let j = 0.9 + (i as f64 * 0.03) % 0.2;
        let sample = IntervalSample {
            seq: i,
            elapsed_sec: 1.0,
            iops: preview.iops * j,
            throughput_mbps: preview.throughput_mbps * j,
            avg_latency_us: None,
        };
        if let Ok(line) = serde_json::to_string(&sample) {
            let _ = std::fs::OpenOptions::new()
                .create(true)
                .append(true)
                .open(&cfg.progress_path)
                .and_then(|mut f| std::io::Write::write_all(&mut f, format!("{line}\n").as_bytes()));
        }
        thread::sleep(tick);
    }
}
