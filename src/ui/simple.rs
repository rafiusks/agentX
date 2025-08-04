use anyhow::Result;
use crossterm::{
    event::{KeyCode, KeyEvent},
    execute,
    style::{Color, ResetColor, SetForegroundColor},
    terminal::{Clear, ClearType},
    cursor,
};
use std::io::{stdout, Write};
use tokio::sync::mpsc;

use super::{LayerType, UILayer};

pub struct SimpleLayer {
    input: String,
    cursor_position: usize,
    is_processing: bool,
    status_message: Option<String>,
    result: Option<String>,
    tx: Option<mpsc::Sender<String>>,
}

impl SimpleLayer {
    pub fn new() -> Self {
        Self {
            input: String::new(),
            cursor_position: 0,
            is_processing: false,
            status_message: None,
            result: None,
            tx: None,
        }
    }
    
    pub fn set_channel(&mut self, tx: mpsc::Sender<String>) {
        self.tx = Some(tx);
    }
    
    pub fn update_result(&mut self, result: String) {
        self.result = Some(result);
        self.is_processing = false;
        self.status_message = Some("✅ Complete! Press Enter to continue.".to_string());
    }

    fn draw_centered_box(&self, width: u16, height: u16) -> Result<()> {
        let term_size = crossterm::terminal::size()?;
        let start_x = (term_size.0 - width) / 2;
        let start_y = (term_size.1 - height) / 2;

        // Clear screen
        execute!(stdout(), Clear(ClearType::All))?;

        // If we have a result, show it instead
        if let Some(result) = &self.result {
            self.draw_result_view(result)?;
            return Ok(());
        }

        // Draw box
        execute!(stdout(), cursor::MoveTo(start_x, start_y))?;
        println!("┌{}┐", "─".repeat((width - 2) as usize));

        // Empty lines
        for i in 1..height-1 {
            execute!(stdout(), cursor::MoveTo(start_x, start_y + i))?;
            println!("│{}│", " ".repeat((width - 2) as usize));
        }

        // Bottom border
        execute!(stdout(), cursor::MoveTo(start_x, start_y + height - 1))?;
        println!("└{}┘", "─".repeat((width - 2) as usize));

        // Draw content
        let content_y = start_y + (height / 2) - 2;
        
        // Title
        let title = "What would you like to create?";
        let title_x = start_x + (width - title.len() as u16) / 2;
        execute!(stdout(), cursor::MoveTo(title_x, content_y))?;
        execute!(stdout(), SetForegroundColor(Color::White))?;
        print!("{}", title);

        // Input field
        let input_y = content_y + 2;
        let input_x = start_x + 4;
        let input_width = (width - 12) as usize;
        
        execute!(stdout(), cursor::MoveTo(input_x, input_y))?;
        print!("[");
        
        // Show input or status
        if self.is_processing {
            execute!(stdout(), SetForegroundColor(Color::Yellow))?;
            print!("{:^width$}", "✨ Working on it...", width = input_width);
        } else {
            execute!(stdout(), SetForegroundColor(Color::White))?;
            print!("{:<width$}", self.input, width = input_width);
        }
        
        execute!(stdout(), ResetColor)?;
        print!("] ");
        
        // Microphone emoji
        execute!(stdout(), SetForegroundColor(Color::Cyan))?;
        print!("🎤");
        execute!(stdout(), ResetColor)?;

        // Quick ideas (if not processing)
        if !self.is_processing && self.input.is_empty() {
            let ideas_y = input_y + 2;
            let ideas = "Quick ideas:";
            let ideas_x = start_x + (width - ideas.len() as u16) / 2;
            execute!(stdout(), cursor::MoveTo(ideas_x, ideas_y))?;
            execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
            print!("{}", ideas);

            let examples = "• Build a web app  • Analyze data  • Write docs";
            let examples_x = start_x + (width - examples.len() as u16) / 2;
            execute!(stdout(), cursor::MoveTo(examples_x, ideas_y + 1))?;
            print!("{}", examples);
            execute!(stdout(), ResetColor)?;
        }

        // Status message
        if let Some(status) = &self.status_message {
            let status_y = start_y + height + 1;
            let status_x = start_x + (width - status.len() as u16) / 2;
            execute!(stdout(), cursor::MoveTo(status_x, status_y))?;
            execute!(stdout(), SetForegroundColor(Color::Green))?;
            print!("{}", status);
            execute!(stdout(), ResetColor)?;
        }

        // Position cursor
        if !self.is_processing {
            let cursor_x = input_x + 1 + self.cursor_position as u16;
            execute!(stdout(), cursor::MoveTo(cursor_x, input_y))?;
            execute!(stdout(), cursor::Show)?;
        } else {
            execute!(stdout(), cursor::Hide)?;
        }

        stdout().flush()?;
        Ok(())
    }

    fn draw_result_view(&self, result: &str) -> Result<()> {
        let term_size = crossterm::terminal::size()?;
        
        // Clear and show result
        execute!(stdout(), Clear(ClearType::All))?;
        execute!(stdout(), cursor::MoveTo(2, 2))?;
        
        // Split result into lines and display
        for (i, line) in result.lines().enumerate() {
            execute!(stdout(), cursor::MoveTo(2, 2 + i as u16))?;
            print!("{}", line);
        }
        
        // Show continue prompt at bottom
        let prompt = "Press Enter to create something else, or Ctrl+Q to quit";
        execute!(stdout(), cursor::MoveTo(2, term_size.1 - 2))?;
        execute!(stdout(), SetForegroundColor(Color::DarkGrey))?;
        print!("{}", prompt);
        execute!(stdout(), ResetColor)?;
        
        execute!(stdout(), cursor::Hide)?;
        stdout().flush()?;
        Ok(())
    }
}

impl UILayer for SimpleLayer {
    fn layer_type(&self) -> LayerType {
        LayerType::Simple
    }

    fn render(&mut self) -> Result<()> {
        self.draw_centered_box(60, 10)
    }

    fn handle_key(&mut self, key: KeyEvent) -> Result<Option<LayerType>> {
        // If showing result, Enter clears it
        if self.result.is_some() && key.code == KeyCode::Enter {
            self.result = None;
            self.status_message = None;
            return Ok(None);
        }

        if self.is_processing {
            return Ok(None);
        }

        match key.code {
            KeyCode::Char(c) => {
                self.input.insert(self.cursor_position, c);
                self.cursor_position += 1;
            }
            KeyCode::Backspace => {
                if self.cursor_position > 0 {
                    self.cursor_position -= 1;
                    self.input.remove(self.cursor_position);
                }
            }
            KeyCode::Left => {
                if self.cursor_position > 0 {
                    self.cursor_position -= 1;
                }
            }
            KeyCode::Right => {
                if self.cursor_position < self.input.len() {
                    self.cursor_position += 1;
                }
            }
            KeyCode::Enter => {
                if !self.input.is_empty() {
                    self.is_processing = true;
                    
                    // Send request to process
                    if let Some(tx) = &self.tx {
                        let _ = tx.try_send(self.input.clone());
                        self.input.clear();
                        self.cursor_position = 0;
                    }
                }
            }
            _ => {}
        }

        Ok(None)
    }
}