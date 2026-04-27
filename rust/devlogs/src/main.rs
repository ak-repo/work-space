use std::net::SocketAddr;
use tracing_subscriber;

mod db;
mod models;
mod routes;
mod services;
mod state;

#[tokio::main]
async fn main() {
    // Initialize tracing
    tracing_subscriber::fmt::init();

    // Set up database
    let pool = db::setup_database().await.unwrap();
    
    // Run migrations
    sqlx::migrate!("./migrations").run(&pool).await.unwrap();

    // Build our application with the required routes
    let app = routes::create_router(pool);

    // Run our app with hyper, listening globally on port 3000
    let addr = SocketAddr::from(([127, 0, 0, 1], 3000));
    tracing::debug!("listening on {}", addr);

    axum::serve(tokio::net::TcpListener::bind(addr).await.unwrap(), app)
        .await
        .unwrap();
}