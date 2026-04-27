use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::error::Error;

#[derive(Debug, Serialize)]
struct GenerateRequest {
    model: String,
    prompt: String,
    stream: bool,
}

#[derive(Debug, Deserialize)]
struct GenerateResponse {
    response: String,
    #[allow(dead_code)]
    done: bool,
}

pub async fn generate_summary(content: &str) -> Result<String, Box<dyn Error>> {
    let client = Client::new();
    
    let request = GenerateRequest {
        model: "llama2".to_string(),
        prompt: format!("Summarize the following text in one sentence:\n\n{}", content),
        stream: false,
    };
    
    let response = client
        .post("http://localhost:11434/api/generate")
        .json(&request)
        .send()
        .await?;
        
    if response.status().is_success() {
        let generate_response: GenerateResponse = response.json().await?;
        Ok(generate_response.response)
    } else {
        Err(format!("Ollama API error: {}", response.status()).into())
    }
}