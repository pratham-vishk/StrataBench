use crate::config::IntervalSample;
use std::fs::OpenOptions;
use std::io::Write;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::thread;
use std::time::{Duration, Instant};

pub fn spawn_progress_writer(
    path: String,
    bucket_ops: Arc<Vec<AtomicU64>>,
    block_size: usize,
    start: Instant,
    duration: Duration,
) -> Option<thread::JoinHandle<()>> {
    if path.is_empty() {
        return None;
    }
    let _ = std::fs::write(&path, "");
    Some(thread::spawn(move || {
        let mut emitted = 0usize;
        while start.elapsed() < duration {
            thread::sleep(Duration::from_millis(500));
            flush_progress(&path, &bucket_ops, block_size, &mut emitted);
        }
        flush_progress(&path, &bucket_ops, block_size, &mut emitted);
    }))
}

fn flush_progress(path: &str, bucket_ops: &[AtomicU64], block_size: usize, emitted: &mut usize) {
    let mut file = match OpenOptions::new().create(true).append(true).open(path) {
        Ok(f) => f,
        Err(_) => return,
    };
    for (i, bucket) in bucket_ops.iter().enumerate().skip(*emitted) {
        let n = bucket.load(Ordering::Relaxed);
        if n == 0 {
            break;
        }
        let sample = IntervalSample {
            seq: (i + 1) as i32,
            elapsed_sec: 1.0,
            iops: n as f64,
            throughput_mbps: n as f64 * block_size as f64 / (1024.0 * 1024.0),
            avg_latency_us: None,
        };
        if let Ok(line) = serde_json::to_string(&sample) {
            let _ = writeln!(file, "{line}");
        }
        *emitted = i + 1;
    }
}
