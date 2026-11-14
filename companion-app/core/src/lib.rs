//! Logic for the Forever Friends companion app that needs no window.
//!
//! No Tauri or GUI dependency, so `cargo test -p forever-friends-core` runs on a machine that has
//! only a Rust toolchain. It reaches the network, the clock and the file system only through what
//! the caller passes in (a root path, a closure or trait the app crate implements).

/// The app's version, from the workspace's `Cargo.toml`.
pub fn version() -> &'static str {
    env!("CARGO_PKG_VERSION")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn version_is_not_empty() {
        assert!(!version().is_empty());
    }
}
