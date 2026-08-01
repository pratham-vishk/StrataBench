#[cfg(target_os = "linux")]
mod common;
#[cfg(target_os = "linux")]
mod io_uring;
#[cfg(target_os = "linux")]
mod pread;

#[cfg(target_os = "linux")]
pub fn run_block(cfg: &crate::config::EngineConfig) -> Result<crate::config::EngineResults, String> {
    if cfg.io_engine().eq_ignore_ascii_case("io_uring") {
        match io_uring::run_block_io_uring(cfg) {
            Ok(res) => return Ok(res),
            Err(e) => eprintln!("stratabench-engine: io_uring failed ({e}), falling back to pread"),
        }
    }
    pread::run_block_pread(cfg)
}

#[cfg(not(target_os = "linux"))]
pub fn run_block(cfg: &crate::config::EngineConfig) -> Result<crate::config::EngineResults, String> {
    let _ = cfg;
    Err("block I/O requires Linux".into())
}
