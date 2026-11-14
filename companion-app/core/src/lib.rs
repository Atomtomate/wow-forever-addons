//! Pure logic for the Forever Friends companion app.
//!
//! No dependencies, no I/O, no GUI: everything here is tested with `cargo test -p
//! forever-friends-core` on a machine that has only a Rust toolchain. Logic that does not need
//! the GUI goes here, not in the app crate.

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
