#[cfg(target_os = "linux")]
mod linux;

#[cfg(target_os = "linux")]
pub use linux::run_block;

#[cfg(not(target_os = "linux"))]
pub fn run_block(cfg: &crate::config::EngineConfig) -> Result<crate::config::EngineResults, String> {
    let _ = cfg;
    Err("block I/O requires Linux".into())
}
