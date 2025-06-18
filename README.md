# AI-Powered Journal App

A next-generation personal journaling application that combines traditional text entry with cutting-edge AI capabilities, semantic search, and intelligent content extraction.

## Why This App is Next-Level

### 🧠 Advanced AI Integration
- **Automatic Content Analysis**: Every journal entry is processed by Qwen 2.5:7b to extract summaries, entities, topics, and sentiment
- **Semantic Understanding**: Uses state-of-the-art nomic-embed-text model to create 768-dimensional embeddings for deep semantic search
- **Intelligent URL Processing**: Automatically fetches and processes linked content through an MCP agent, enriching your entries with relevant context

### 🔍 Three Revolutionary Search Modes
1. **Classic Search**: Traditional keyword search with powerful filters for favorites, collections, and date ranges
2. **Vector Search**: Pure AI-powered semantic search that understands meaning, not just keywords
   - Find similar thoughts and experiences
   - Explore conceptual relationships
   - Discover contrasting viewpoints
3. **Hybrid Search**: The best of both worlds - combines traditional search precision with AI understanding
   - Smart filters that adapt to your content
   - AI-powered suggestions
   - Temporal pattern recognition

### 📊 Unique Features
- **Temporal Tracking**: Preserves original entries when updated, allowing you to track how your thoughts evolve
- **Asynchronous Processing**: Entries save instantly while AI processing happens in the background
- **Collection Organization**: Visual, Pinterest-style layout with newest entries flowing left-to-right
- **Real-time Updates**: All search and filter changes happen instantly without page refreshes
- **Smart Metadata**: Extracted entities, topics, and sentiment enhance searchability and insights

## System Requirements

### Required Software
- **macOS** (tested on macOS 15 Sequoia)
- **PostgreSQL 16** with pgvector extension
- **Go 1.24** or higher
- **Node.js 24.2.0** or higher with npm 11.3.0+
- **Ollama** (for running AI models locally)
- **Homebrew** (for installing dependencies)

### Hardware Requirements
- **RAM**: Minimum 16GB (32GB recommended for optimal AI performance)
- **Storage**: At least 20GB free space for models and data
- **CPU**: Apple Silicon (M1/M2/M3) or Intel i7+ recommended
- **GPU**: Not required (Ollama uses CPU optimizations)

### Required Models
- **Qwen 2.5:7b** (~4.7GB) - For content processing
- **nomic-embed-text** (~275MB) - For embeddings

## Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd journal
   ```

2. **Install system dependencies**
   ```bash
   brew install postgresql@16 pgvector ollama
   brew services start postgresql@16
   ```

3. **Set up the database**
   ```bash
   createdb journal_db
   make db-setup
   make db-migrate
   ```

4. **Install Ollama models**
   ```bash
   ollama pull qwen2.5:7b
   ollama pull nomic-embed-text
   ```

5. **Install application dependencies**
   ```bash
   make install-deps
   ```

6. **Set environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   export DB_USER=$USER  # Use your macOS username
   ```

7. **Start the application**
   ```bash
   make dev
   ```

## Architecture

### Backend (Go)
- **JSON-RPC API**: Clean, structured API design
- **PostgreSQL + pgvector**: Combines relational data with vector embeddings
- **Asynchronous Processing**: Non-blocking AI operations
- **MCP Integration**: Model Context Protocol for extensible AI tools

### Frontend (React)
- **Vite + React**: Lightning-fast development experience
- **Tailwind CSS v4**: Modern, utility-first styling
- **TanStack Query**: Intelligent data fetching and caching
- **Real-time Updates**: Reactive UI with instant feedback

### AI Pipeline
1. User creates journal entry
2. Entry saves immediately to database
3. Background processing:
   - Qwen analyzes content for insights
   - MCP agent fetches linked content
   - Embeddings generated for semantic search
   - Database updated with enriched data

## Key Innovations

### Temporal Embeddings
When entries are updated, the original embedding is preserved, allowing you to track how your understanding and expression of ideas evolves over time.

### Hybrid Search Intelligence
The hybrid search mode doesn't just combine results - it uses AI to understand the relationship between your search intent and available filters, providing contextual suggestions.

### Asynchronous AI Processing
Unlike traditional apps that make you wait, entries save instantly while AI enrichment happens seamlessly in the background.

### MCP Agent Architecture
The modular MCP agent design allows for future expansion - add new tools for weather data, calendar integration, or any other contextual information.

## New Features (Phase 1-3 Implementation)

### Enhanced Search Capabilities
- **Vector Search Modes**: Similar (default), Explore (conceptual connections), and Contrast (opposing viewpoints)
- **Hybrid Search Strategies**: Balanced, Semantic Boost, Precision Mode, and Discovery Mode with weighted scoring
- **Search Suggestions**: Popular topics, entities, and recent entries displayed when search is empty
- **Full Filtering**: All search types now support collections, favorites, and date filtering

### Improved User Experience
- **Retry Processing**: Failed entries can be retried with automatic state reset
- **Keyboard Shortcuts**: Comprehensive shortcuts with help panel (press ? to view)
  - Ctrl+N: New entry
  - Ctrl+K: Focus search
  - Ctrl+E: Edit entry
  - Ctrl+S: Save entry
  - Ctrl+Enter: Toggle fullscreen
- **Export Functionality**: Export entries in JSON, Markdown, or CSV formats
- **Enhanced Error Handling**: Global error boundary, detailed error messages, and retry logic
- **Advanced Processing Tracker**: Domino's-inspired visual progress tracker with:
  - Real-time stage updates with animated icons
  - Auto-collapse after 2 seconds on completion
  - Manual expand/collapse controls
  - Clickable AI Analysis box to toggle visibility
  - Different visual states for processing vs completed entries

### AI Enhancements
- **Improved Prompts**: All Ollama prompts now include few-shot examples for better consistency
- **Smart Failure Analysis**: AI-powered analysis of processing failures with actionable solutions
- **Richer Embeddings**: Metadata included in embeddings for better semantic search

### Real-Time Updates
- **SSE Events**: All collection operations now trigger real-time updates
- **Live Processing Status**: Timer updates every second during processing
- **Optimistic UI**: Immediate feedback for user actions

## Development

### Running Tests
```bash
make test
```

### Building for Production
```bash
make build
```

### Viewing Logs
```bash
make tail-log
```

## Troubleshooting

### Database Connection Issues
Ensure PostgreSQL is running and your user has proper permissions:
```bash
psql -U $USER -d postgres -c "SELECT current_user;"
```

### Model Download Issues
If Ollama models fail to download, ensure Ollama is running:
```bash
ollama serve
```

### Port Conflicts
If ports are already in use:
```bash
lsof -ti:8080 | xargs kill -9  # Backend
lsof -ti:5173 | xargs kill -9  # Frontend
```

## Future Enhancements
- Voice-to-text journal entries
- Multi-modal embeddings (images, audio)
- Collaborative collections
- Mobile applications
- Advanced visualization of thought patterns
- Batch import from other journaling apps
- Time-based analytics and insights
- Custom AI models for specialized use cases

## License
[License Type]

## Contributing
[Contribution guidelines]