use agentx::intelligence::{AISystem, ErrorContext};
use anyhow::Result;
use std::path::Path;
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<()> {
    println!("🤖 AgentX AI Features Demo\n");
    
    // Initialize AI system
    let ai_system = AISystem::new();
    
    // Demo 1: Natural Language Processing
    demo_nlp(&ai_system).await?;
    
    // Demo 2: Intelligent Command Suggestions
    demo_suggestions(&ai_system).await?;
    
    // Demo 3: Error Diagnosis
    demo_error_diagnosis(&ai_system).await?;
    
    // Demo 4: Context Management
    demo_context_management(&ai_system).await?;
    
    // Demo 5: Adaptive UI
    demo_adaptive_ui(&ai_system).await?;
    
    Ok(())
}

async fn demo_nlp(ai: &AISystem) -> Result<()> {
    println!("=== Natural Language Processing Demo ===\n");
    
    let queries = vec![
        "create a new file called test.rs",
        "run all tests",
        "build the project",
        "commit changes with message 'Initial commit'",
        "search for TODO in all rust files",
    ];
    
    for query in queries {
        println!("Query: \"{}\"", query);
        
        match ai.process_query(query).await {
            Ok(commands) => {
                println!("Generated commands:");
                for cmd in commands {
                    println!("  $ {} {}", cmd.command, cmd.args.join(" "));
                    if !cmd.description.is_empty() {
                        println!("    ({})", cmd.description);
                    }
                }
            }
            Err(e) => println!("  Error: {}", e),
        }
        println!();
    }
    
    Ok(())
}

async fn demo_suggestions(ai: &AISystem) -> Result<()> {
    println!("\n=== Intelligent Command Suggestions Demo ===\n");
    
    // Simulate command history
    let command_history = vec![
        ("cargo test", true, Duration::from_secs(2)),
        ("cargo build --release", true, Duration::from_secs(10)),
        ("git add .", true, Duration::from_millis(100)),
        ("git commit -m 'Fix bug'", true, Duration::from_secs(1)),
        ("cargo test", true, Duration::from_secs(2)),
        ("npm install", false, Duration::from_secs(5)),
    ];
    
    // Record command history
    for (cmd, success, duration) in command_history {
        ai.record_command(cmd.to_string(), success, duration).await;
    }
    
    // Test suggestions
    let test_inputs = vec!["", "car", "git", "test"];
    
    for input in test_inputs {
        println!("Input: \"{}\"", input);
        let suggestions = ai.get_suggestions(input).await;
        
        if suggestions.is_empty() {
            println!("  No suggestions");
        } else {
            println!("  Suggestions:");
            for (i, suggestion) in suggestions.iter().take(3).enumerate() {
                println!("    {}. {} (confidence: {:.2})", 
                    i + 1, 
                    suggestion.command, 
                    suggestion.confidence
                );
                println!("       {}", suggestion.explanation);
                if let Some(ref shortcut) = suggestion.keyboard_shortcut {
                    println!("       Shortcut: {}", shortcut);
                }
                if let Some(ref next_cmds) = suggestion.next_commands {
                    println!("       Next: {}", next_cmds.join(" → "));
                }
            }
        }
        println!();
    }
    
    Ok(())
}

async fn demo_error_diagnosis(ai: &AISystem) -> Result<()> {
    println!("\n=== Error Diagnosis Demo ===\n");
    
    let error_scenarios = vec![
        (
            "Error: listen EADDRINUSE: address already in use :::3000",
            ErrorContext {
                command: "npm start".to_string(),
                working_directory: "/project".to_string(),
                environment: std::collections::HashMap::new(),
                exit_code: 1,
            },
        ),
        (
            "Error: Cannot find module 'express'",
            ErrorContext {
                command: "node server.js".to_string(),
                working_directory: "/app".to_string(),
                environment: std::collections::HashMap::new(),
                exit_code: 1,
            },
        ),
        (
            "error[E0433]: failed to resolve: use of undeclared type `Vec`",
            ErrorContext {
                command: "cargo build".to_string(),
                working_directory: "/rust-project".to_string(),
                environment: std::collections::HashMap::new(),
                exit_code: 101,
            },
        ),
    ];
    
    for (error_output, context) in error_scenarios {
        println!("Error Output:\n{}\n", error_output);
        
        match ai.diagnose_error(error_output, &context).await {
            Ok(diagnosis) => {
                println!("Diagnosis:");
                println!("  Type: {:?}", diagnosis.error_type);
                println!("  Severity: {:?}", diagnosis.severity);
                println!("  Explanation: {}", diagnosis.explanation);
                println!("  Root Cause: {}", diagnosis.root_cause);
                
                println!("\n  Suggested Fixes:");
                for (i, fix) in diagnosis.suggested_fixes.iter().take(2).enumerate() {
                    println!("    {}. {} (confidence: {:.2})", 
                        i + 1, 
                        fix.description, 
                        fix.confidence
                    );
                    for cmd in &fix.commands {
                        println!("       $ {}", cmd);
                    }
                    if !fix.side_effects.is_empty() {
                        println!("       Side effects: {}", fix.side_effects.join(", "));
                    }
                }
                
                if !diagnosis.prevention_tips.is_empty() {
                    println!("\n  Prevention Tips:");
                    for tip in diagnosis.prevention_tips.iter().take(2) {
                        println!("    • {}", tip);
                    }
                }
            }
            Err(e) => println!("  Diagnosis failed: {}", e),
        }
        
        println!("\n{}", "─".repeat(60));
        println!();
    }
    
    Ok(())
}

async fn demo_context_management(ai: &AISystem) -> Result<()> {
    println!("\n=== Context Management Demo ===\n");
    
    // Get context for current directory
    let current_dir = std::env::current_dir()?;
    
    match ai.build_context(&current_dir).await {
        Ok(context) => {
            println!("Developer Context:");
            println!("  Project: {}", context.project_name);
            println!("  Type: {:?}", context.project_type);
            println!("  Root: {}", context.project_root.display());
            
            if !context.languages.is_empty() {
                println!("\n  Languages detected:");
                for lang in &context.languages {
                    println!("    • {:?}", lang);
                }
            }
            
            if !context.frameworks.is_empty() {
                println!("\n  Frameworks detected:");
                for framework in &context.frameworks {
                    println!("    • {:?}", framework);
                }
            }
            
            if let Some(ref git_info) = context.git_info {
                println!("\n  Git Information:");
                println!("    Branch: {}", git_info.current_branch);
                if let Some(ref remote) = git_info.remote_url {
                    println!("    Remote: {}", remote);
                }
                println!("    Uncommitted changes: {}", git_info.uncommitted_changes.len());
                println!("    Stashes: {}", git_info.stash_count);
            }
            
            if !context.installed_tools.is_empty() {
                println!("\n  Development Tools:");
                for tool in context.installed_tools.iter().take(5) {
                    println!("    • {}: {}", tool.name, tool.version);
                }
            }
            
            println!("\n  Environment variables: {} (filtered for privacy)", 
                context.environment_vars.len());
            
            if !context.related_repos.is_empty() {
                println!("\n  Related repositories in workspace:");
                for repo in &context.related_repos {
                    println!("    • {} ({:?})", repo.name, repo.main_language);
                }
            }
        }
        Err(e) => println!("  Context building failed: {}", e),
    }
    
    Ok(())
}

async fn demo_adaptive_ui(ai: &AISystem) -> Result<()> {
    println!("\n\n=== Adaptive UI Demo ===\n");
    
    // Simulate user interactions
    println!("Simulating user journey...\n");
    
    // Initial state
    let initial_expertise = ai.get_user_expertise().await;
    println!("Initial expertise level: {:.2} (Beginner)", initial_expertise);
    
    // Simulate interactions
    for i in 1..=15 {
        ai.record_interaction().await;
        
        if i % 5 == 0 {
            let expertise = ai.get_user_expertise().await;
            println!("After {} interactions: expertise = {:.2}", i, expertise);
            
            match expertise {
                e if e < 0.2 => println!("  → UI Mode: Simple (consumer-friendly)"),
                e if e < 0.5 => println!("  → UI Mode: Power (showing more features)"),
                e if e < 0.8 => println!("  → UI Mode: Pro (full control available)"),
                _ => println!("  → UI Mode: Expert (all features visible)"),
            }
        }
    }
    
    // Simulate layer transitions
    println!("\nSimulating direct layer transitions:");
    
    use agentx::ui::LayerType;
    
    ai.record_layer_transition(LayerType::Power).await;
    let expertise = ai.get_user_expertise().await;
    println!("  Switched to Power mode → expertise: {:.2}", expertise);
    
    ai.record_layer_transition(LayerType::Pro).await;
    let expertise = ai.get_user_expertise().await;
    println!("  Switched to Pro mode → expertise: {:.2}", expertise);
    
    println!("\n✨ The UI would now show advanced features and shortcuts!");
    
    Ok(())
}