// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use anyhow::Result;

fn main() -> Result<()> {
    // Check if running in Tauri mode
    if std::env::var("TAURI").is_ok() || std::env::args().any(|arg| arg == "--tauri") {
        #[cfg(feature = "tauri")]
        {
            agentx_lib::tauri_main::run();
            Ok(())
        }
        #[cfg(not(feature = "tauri"))]
        {
            eprintln!("Tauri feature not enabled. Build with --features tauri");
            std::process::exit(1);
        }
    } else {
        // Run terminal mode
        tokio::runtime::Runtime::new()?.block_on(async {
            // Initialize tracing
            tracing_subscriber::fmt()
                .with_target(false)
                .with_level(true)
                .init();

            // Start AgentX in terminal mode
            agentx_lib::run().await
        })
    }
}