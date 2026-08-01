use rand::Rng;
use std::os::unix::fs::FileExt;
use std::sync::Arc;
use std::thread;

use super::common::{
    aligned_buffer, open_target, pick_offset, should_write, spawn_progress, validate_block_cfg,
    RunState,
};

pub fn run_block_pread(cfg: &crate::config::EngineConfig) -> Result<crate::config::EngineResults, String> {
    let (bs, dataset) = validate_block_cfg(cfg)?;
    let file = open_target(&cfg.target, cfg.direct_io)?;
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
        let state = Arc::clone(&state);
        let pattern = cfg.pattern.clone();
        let mix = cfg.read_write_mix;
        let duration = state.duration;
        let start = state.start;

        handles.push(thread::spawn(move || {
            let mut buf = aligned_buffer(bs, 4096);
            let mut rng = rand::thread_rng();
            let mut seq_off = (tid as u64) * bs as u64;
            while start.elapsed() < duration {
                let write = should_write(&pattern, mix, &mut rng);
                let offset = pick_offset(&pattern, dataset, bs, &mut seq_off, tid, &mut rng);
                let op_start = std::time::Instant::now();
                let ok = if write {
                    file.write_at(&buf, offset).is_ok()
                } else {
                    file.read_at(&mut buf, offset).is_ok()
                };
                if !ok {
                    continue;
                }
                let us = op_start.elapsed().as_secs_f64() * 1_000_000.0;
                state.record(write, us);
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
