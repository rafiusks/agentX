use anyhow::Result;
use crossterm::event::{self, Event, KeyCode, KeyEvent};
use std::time::Duration;
use tokio::sync::mpsc;
use std::sync::Arc;
use tokio::sync::Mutex;

use crate::ui::{UILayer, LayerType, simple::SimpleLayer};
use crate::intelligence::UserContext;
use crate::agents::orchestrator::Orchestrator;

pub struct Application {
    current_layer: LayerType,
    user_context: UserContext,
    simple_layer: SimpleLayer,
    should_quit: bool,
    _request_tx: mpsc::Sender<String>,
    request_rx: mpsc::Receiver<String>,
    result_tx: mpsc::Sender<String>,
    result_rx: mpsc::Receiver<String>,
}

impl Application {
    pub fn new() -> Result<Self> {
        // Create channels for async communication
        let (request_tx, request_rx) = mpsc::channel(100);
        let (result_tx, result_rx) = mpsc::channel(100);
        
        // Create simple layer and give it the sender
        let mut simple_layer = crate::ui::create_simple_layer();
        simple_layer.set_channel(request_tx.clone());
        
        Ok(Self {
            current_layer: LayerType::Simple,
            user_context: UserContext::new(),
            simple_layer,
            should_quit: false,
            _request_tx: request_tx,
            request_rx,
            result_tx,
            result_rx,
        })
    }

    pub async fn run(&mut self) -> Result<()> {
        // Initialize terminal
        self.setup_terminal()?;
        
        // Start background task processor
        let result_tx = self.result_tx.clone();
        let mut request_rx = std::mem::replace(&mut self.request_rx, mpsc::channel(1).1);
        
        tokio::spawn(async move {
            let orchestrator = Arc::new(Mutex::new(Orchestrator::new()));
            
            while let Some(request) = request_rx.recv().await {
                let mut orch = orchestrator.lock().await;
                match orch.process_request(&request).await {
                    Ok(result) => {
                        let _ = result_tx.send(result).await;
                    }
                    Err(e) => {
                        let _ = result_tx.send(format!("❌ Error: {}", e)).await;
                    }
                }
            }
        });
        
        // Main event loop
        let result = self.main_loop().await;
        
        // Cleanup
        self.restore_terminal()?;
        
        result
    }

    async fn main_loop(&mut self) -> Result<()> {
        loop {
            // Check for results from background processing
            if let Ok(result) = self.result_rx.try_recv() {
                self.simple_layer.update_result(result);
                self.user_context.record_interaction();
            }
            
            // Render current layer
            self.render()?;
            
            // Handle events with timeout to allow checking for results
            if self.handle_events_with_timeout().await? {
                break;
            }
            
            // Check if we should transition layers
            if let Some(new_layer) = self.check_layer_transition() {
                self.transition_to(new_layer);
            }
        }
        
        Ok(())
    }

    fn render(&mut self) -> Result<()> {
        match self.current_layer {
            LayerType::Simple => self.simple_layer.render()?,
            _ => {} // Other layers not implemented yet
        }
        Ok(())
    }

    async fn handle_events_with_timeout(&mut self) -> Result<bool> {
        // Poll for events with small timeout to stay responsive
        if event::poll(Duration::from_millis(50))? {
            if let Event::Key(key) = event::read()? {
                // Global shortcuts
                if self.handle_global_shortcuts(&key) {
                    return Ok(self.should_quit);
                }
                
                // Pass to current layer
                match self.current_layer {
                    LayerType::Simple => {
                        if let Some(transition) = self.simple_layer.handle_key(key)? {
                            self.transition_to(transition);
                        }
                    }
                    _ => {} // Other layers not implemented yet
                }
            }
        }
        
        Ok(self.should_quit)
    }

    fn handle_global_shortcuts(&mut self, key: &KeyEvent) -> bool {
        match key.code {
            KeyCode::Char('q') if key.modifiers.contains(crossterm::event::KeyModifiers::CONTROL) => {
                self.should_quit = true;
                true
            }
            KeyCode::Char('p') if key.modifiers.contains(crossterm::event::KeyModifiers::CONTROL | crossterm::event::KeyModifiers::SHIFT) => {
                // Toggle pro mode
                self.current_layer = if self.current_layer == LayerType::Pro {
                    LayerType::Simple
                } else {
                    LayerType::Pro
                };
                true
            }
            _ => false
        }
    }

    fn check_layer_transition(&self) -> Option<LayerType> {
        // Progressive disclosure logic
        if self.user_context.interaction_count > 5 && self.current_layer == LayerType::Simple {
            return Some(LayerType::Mission);
        }
        None
    }

    fn transition_to(&mut self, new_layer: LayerType) {
        self.current_layer = new_layer;
        self.user_context.record_transition(new_layer);
    }

    fn setup_terminal(&self) -> Result<()> {
        crossterm::terminal::enable_raw_mode()?;
        crossterm::execute!(
            std::io::stdout(),
            crossterm::terminal::EnterAlternateScreen,
            crossterm::event::EnableMouseCapture
        )?;
        Ok(())
    }

    fn restore_terminal(&self) -> Result<()> {
        crossterm::terminal::disable_raw_mode()?;
        crossterm::execute!(
            std::io::stdout(),
            crossterm::terminal::LeaveAlternateScreen,
            crossterm::event::DisableMouseCapture
        )?;
        Ok(())
    }
}