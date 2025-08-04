use crossterm::{
    execute,
    terminal::{self, EnterAlternateScreen, LeaveAlternateScreen},
    event::{self, Event, KeyCode, KeyModifiers},
    style::{Color, ResetColor, SetForegroundColor},
    cursor,
};
use std::io::{stdout, Write};
use std::time::Duration;
use anyhow::Result;

fn main() -> Result<()> {
    println!("🚀 Simple Terminal Test");
    println!("Testing basic terminal capabilities...");
    
    // Test 1: Basic terminal size
    match terminal::size() {
        Ok(size) => println!("✅ Terminal size: {}x{}", size.0, size.1),
        Err(e) => {
            println!("❌ Cannot get terminal size: {}", e);
            return Ok(());
        }
    }
    
    // Test 2: Basic color output
    println!("✅ Basic color test:");
    execute!(stdout(), SetForegroundColor(Color::Green))?;
    print!("Green text ");
    execute!(stdout(), SetForegroundColor(Color::Red))?;
    print!("Red text ");
    execute!(stdout(), ResetColor)?;
    println!("Normal text");
    
    // Test 3: Try raw mode briefly (this is where issues might occur)
    println!("\n🔧 Testing raw mode (press any key, or wait 3 seconds)...");
    
    match terminal::enable_raw_mode() {
        Ok(_) => {
            println!("✅ Raw mode enabled");
            
            // Test event polling
            let mut found_input = false;
            for _ in 0..30 { // 3 seconds with 100ms intervals
                if event::poll(Duration::from_millis(100))? {
                    if let Event::Key(key) = event::read()? {
                        println!("✅ Key detected: {:?}", key);
                        found_input = true;
                        break;
                    }
                }
            }
            
            if !found_input {
                println!("⚠️  No key pressed within 3 seconds");
            }
            
            terminal::disable_raw_mode()?;
            println!("✅ Raw mode disabled");
        }
        Err(e) => {
            println!("❌ Cannot enable raw mode: {}", e);
            println!("   This is likely the source of the 'Device not configured' error");
            return Ok(());
        }
    }
    
    // Test 4: Try alternate screen (another potential issue)
    println!("\n🔧 Testing alternate screen...");
    
    match execute!(stdout(), EnterAlternateScreen) {
        Ok(_) => {
            // Clear and draw something
            execute!(stdout(), terminal::Clear(terminal::ClearType::All))?;
            execute!(stdout(), cursor::MoveTo(10, 5))?;
            execute!(stdout(), SetForegroundColor(Color::Cyan))?;
            print!("🎯 This is the alternate screen!");
            execute!(stdout(), ResetColor)?;
            execute!(stdout(), cursor::MoveTo(10, 7))?;
            print!("Press any key to return...");
            stdout().flush()?;
            
            // Wait for a key
            terminal::enable_raw_mode()?;
            loop {
                if event::poll(Duration::from_millis(100))? {
                    if let Event::Key(_) = event::read()? {
                        break;
                    }
                }
            }
            terminal::disable_raw_mode()?;
            
            execute!(stdout(), LeaveAlternateScreen)?;
            println!("✅ Alternate screen test passed");
        }
        Err(e) => {
            println!("❌ Cannot use alternate screen: {}", e);
        }
    }
    
    println!("\n🎉 Terminal tests completed!");
    println!("If all tests passed, the terminal UI should work.");
    
    Ok(())
}