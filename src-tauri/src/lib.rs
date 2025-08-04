pub mod app;
pub mod ui;
pub mod agents;
pub mod intelligence;
pub mod infrastructure;
pub mod providers;
pub mod config;
pub mod mcp;

#[cfg(feature = "tauri")]
pub mod tauri_main;

use anyhow::Result;

pub async fn run() -> Result<()> {
    // Start with the simplest layer
    let mut app = app::Application::new()?;
    app.run().await
}