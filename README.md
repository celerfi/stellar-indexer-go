# CelerFi Stellar Indexer

The CelerFi Stellar Indexer is a high-performance data ingestion and analytics engine designed to index the Stellar blockchain (Soroban and Classic) and expose real-time DeFi metrics via a unified GraphQL interface. It utilizes TimescaleDB for optimized time-series storage and provides automated calculations for Total Value Locked (TVL), 24-hour volume, and price ticks.

## Core Capabilities

*   **Protocol Support**: Native indexing for Soroswap, Aquarius, and the Blend Lending Protocol.
*   **Pricing Engine**: Real-time USD price derivation from on-chain Reflector oracles and AMM swap ratios.
*   **Time-Series Analytics**: Automated generation of OHLCV data at 1m, 1h, and 1d intervals using TimescaleDB.
*   **Global Transfer Tracking**: Monitoring of all asset movements, including native XLM and SAC tokens.
*   **Unified API**: Standardized GraphQL layer for querying transactions, tokens, pools, and analytics.

## Technology Stack

*   **Language**: Go 1.25+
*   **Database**: PostgreSQL with TimescaleDB extension
*   **API**: GraphQL (via gqlgen)
*   **Orchestration**: Docker for containerized database environments

## Getting Started

### Prerequisites

*   Go 1.25 or higher
*   Docker and Docker Compose
*   Node.js/pnpm (for workflow management)

### Environment Configuration

Create a `dev.env` file in the root directory with the following variables:

```env
DB_USER=eren
DB_PASSWORD=mikasa_scarf
DB_HOST=localhost
DB_NAME=stellar_indexer
RPC_URL=your_soroban_rpc_url
HORIZON_URL=your_horizon_url
DEPLOYMENT_ENVIRONMENT=production
```

### Installation and Execution

1. Initialize the database environment:
   ```bash
   pnpm db:start
   pnpm db:init # you'd wait for about 10 secs for the db to startup before initializing
   ```

2. Build the application:
   ```bash
   pnpm build
   ```

3. Run the indexer and API server:
   ```bash
   pnpm dev
   ```

## Development Workflow

*   **Database Inspection**: 
    *   `pnpm inspect:trades`: View recent transactions
    *   `pnpm inspect:prices`: View recent price ticks
    *   `pnpm inspect:tvl`: View current pool TVL
*   **API Testing**: Access the interactive GraphQL playground at `http://localhost:8080` when the server is running

## Database Schema

The system relies on several optimized SQL schemas:
*   `main.sql`: Core transaction and operation storage.
*   `price_ticks.sql`: Time-series price data using hypertables.
*   `analytics_views.sql`: Materialized views for DeFi metrics.
*   `blend_events.sql`: Lending protocol specific events.
*   `liquidity_actions.sql`: AMM deposit and withdrawal tracking.
