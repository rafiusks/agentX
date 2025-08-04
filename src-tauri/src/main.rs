// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    #[cfg(feature = "tauri")]
    {
        agentx_lib::tauri_main::run();
    }
    
    #[cfg(not(feature = "tauri"))]
    {
        // Run terminal mode
        tokio::runtime::Runtime::new().unwrap().block_on(async {
            // Initialize tracing
            tracing_subscriber::fmt()
                .with_target(false)
                .with_level(true)
                .init();

            // Start AgentX in terminal mode
            agentx_lib::run().await.unwrap();
        });
    }
}