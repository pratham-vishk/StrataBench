use io_uring::{opcode, types, IoUring};
use rand::Rng;
use std::os::unix::io::AsRawFd;
use std::sync::Arc;
use std::thread;
use std::time::Instant;

use super::common::{
    aligned_buffer, open_target, pick_offset, should_write, spawn_progress, validate_block_cfg,
    RunState,
};

struct InflightOp {
    started: Instant,
    write: bool,
}

pub fn run_block_io_uring(
    cfg: &crate::config::EngineConfig,
) -> Result<crate::config::EngineResults, String> {
    let (bs, dataset) = validate_block_cfg(cfg)?;
    let file = open_target(&cfg.target, cfg.direct_io)?;
    let qd = cfg.queue_depth.max(1) as usize;
    let ring_entries = (qd * 2).max(8) as u32;
    let state = Arc::new(RunState::new(cfg, bs));
    let progress = spawn_progress(
        cfg,
        Arc::clone(&state.bucket_ops),
        bs,
        state.start,
        state.duration,
    );
    let threads = cfg.threads.max(1) as usize;
    let mut handles = Vec::with_capacity(threads);

    for tid in 0..threads {
        let file = file.try_clone().map_err(|e| e.to_string())?;
        let worker_fd = types::Fd(file.as_raw_fd());
        let state = Arc::clone(&state);
        let pattern = cfg.pattern.clone();
        let mix = cfg.read_write_mix;
        let duration = state.duration;
        let start = state.start;

        handles.push(thread::spawn(move || {
            let mut ring = match IoUring::new(ring_entries) {
                Ok(r) => r,
                Err(e) => {
                    eprintln!("stratabench-engine: io_uring init failed: {e}");
                    return;
                }
            };
            let mut buffers: Vec<Vec<u8>> = (0..qd).map(|_| aligned_buffer(bs, 4096)).collect();
            let mut inflight: Vec<Option<InflightOp>> = vec![None; qd];
            let mut rng = rand::thread_rng();
            let mut seq_off = (tid as u64) * bs as u64;
            let mut pending = 0usize;

            while start.elapsed() < duration || pending > 0 {
                while pending < qd && start.elapsed() < duration {
                    let slot = match inflight.iter().position(|s| s.is_none()) {
                        Some(i) => i,
                        None => break,
                    };
                    let write = should_write(&pattern, mix, &mut rng);
                    let offset =
                        pick_offset(&pattern, dataset, bs, &mut seq_off, tid, &mut rng);
                    let buf = buffers[slot].as_mut_slice();
                    let sqe = if write {
                        opcode::Write::new(worker_fd, buf.as_ptr(), bs as u32)
                            .offset(offset as i64)
                            .user_data(slot as u64)
                            .build()
                    } else {
                        opcode::Read::new(worker_fd, buf.as_ptr(), bs as u32)
                            .offset(offset as i64)
                            .user_data(slot as u64)
                            .build()
                    };
                    let pushed = unsafe {
                        let mut sq = ring.submission();
                        sq.push(&sqe).is_ok()
                    };
                    if !pushed {
                        break;
                    }
                    inflight[slot] = Some(InflightOp {
                        started: Instant::now(),
                        write,
                    });
                    pending += 1;
                }

                if pending == 0 {
                    continue;
                }

                if ring.submit().is_err() {
                    continue;
                }

                let mut reaped = 0usize;
                while let Some(cqe) = ring.completion().next() {
                    reaped += 1;
                    let slot = cqe.user_data() as usize;
                    if slot < inflight.len() {
                        if cqe.result() >= 0 {
                            if let Some(op) = inflight[slot].take() {
                                let us = op.started.elapsed().as_secs_f64() * 1_000_000.0;
                                state.record(op.write, us);
                            }
                        } else {
                            inflight[slot] = None;
                        }
                    }
                    pending = pending.saturating_sub(1);
                }

                if reaped == 0 {
                    let _ = ring.submit_and_wait(1);
                }
            }
        }));
    }

    for h in handles {
        h.join().map_err(|_| "worker panicked".to_string())?;
    }
    if let Some(h) = progress {
        let _ = h.join();
    }
    Ok(state.finish())
}

#[cfg(test)]
mod tests {
    #[test]
    fn ring_entries_headroom() {
        let qd = 32usize;
        let entries = (qd * 2).max(8) as u32;
        assert!(entries >= qd as u32);
    }
}
