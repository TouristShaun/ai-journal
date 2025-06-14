# Lessons Learned: Building an AI-Powered Journal App

## Technical Insights

### 1. Asynchronous AI Processing is Critical
**Problem**: Initial implementation processed AI analysis synchronously, causing the app to crash when Ollama took too long to respond.

**Solution**: Implemented async processing where entries save immediately and AI enrichment happens in background goroutines.

**Key Learning**: User experience should never be blocked by AI processing. Always provide immediate feedback and handle enrichment asynchronously.

### 2. Embedding Response Formats Vary
**Problem**: Ollama's embedding API returns nested arrays `[[...embeddings...]]` not flat arrays, causing JSON unmarshaling errors.

**Solution**: Updated the struct to handle `[][]float32` and extract the first embedding array.

**Key Learning**: Always test AI API responses thoroughly - documentation may not reflect actual response formats.

### 3. Tailwind CSS v4 Changes Everything
**Problem**: PostCSS configuration that worked in v3 broke completely in v4.

**Solution**: Tailwind v4 uses a dedicated Vite plugin and single `@import "tailwindcss"` directive.

**Key Learning**: Major version updates often bring architectural changes. The new approach is actually simpler and faster.

### 4. Database User Permissions Matter
**Problem**: Default PostgreSQL installations don't always create a "postgres" user, causing connection failures.

**Solution**: Use the system user (`$USER`) for local development instead of hardcoding "postgres".

**Key Learning**: Make database configuration flexible and document system-specific requirements clearly.

### 5. pgvector Installation Complexity
**Problem**: Homebrew installs pgvector for different PostgreSQL versions, causing "extension not found" errors.

**Solution**: Built pgvector from source specifically for PostgreSQL 16.

**Key Learning**: Extension compatibility is version-specific. Sometimes building from source is more reliable than package managers.

## Architecture Decisions

### 1. JSON-RPC Over REST
**Why**: Provides a clean, structured API with consistent error handling and method naming.

**Benefit**: Easier to add new methods without worrying about HTTP verbs and URL structures.

**Trade-off**: Less familiar to developers used to REST, but the consistency pays off.

### 2. Go Structs Instead of ORM
**Why**: Following the principle of "the dumbest possible thing that works".

**Benefit**: Direct SQL queries are easier to debug and match database logs exactly.

**Trade-off**: More boilerplate code, but much clearer data flow.

### 3. Three Distinct Search Modes
**Why**: Each mode serves a different mental model - sometimes you know what you want (classic), sometimes you're exploring (vector), sometimes both (hybrid).

**Benefit**: Users can choose the right tool for their current need instead of a one-size-fits-all approach.

**Trade-off**: More complex UI, but the clarity of purpose justifies it.

### 4. MCP for URL Fetching
**Why**: Separates concerns and makes the system extensible for future tools.

**Benefit**: Can add new capabilities (weather, calendar, etc.) without modifying core journal logic.

**Trade-off**: Additional service to maintain, but the modularity is worth it.

## UI/UX Insights

### 1. Instant Feedback is Non-Negotiable
**Learning**: Users need to see their entry saved immediately. Background processing should be invisible until complete.

### 2. Color Consistency Matters
**Learning**: Using Tailwind's dynamic class names (like `bg-${color}-100`) doesn't work reliably. Explicit classes are better.

### 3. Error States Need Clear Communication
**Learning**: Silent failures are the worst UX. Always show loading states, error messages, and recovery options.

### 4. Search Mode Visualization
**Learning**: Icons and descriptions help users understand the difference between search modes at a glance.

## Performance Optimizations

### 1. Embedding Generation Timing
**Learning**: Generate embeddings AFTER fetching URL content for richer semantic representation.

### 2. Database Indexing Strategy
- GIN index for full-text search
- IVFFlat index for vector similarity (with 100 lists for ~10k entries)
- B-tree indexes for timestamp queries

### 3. Query Optimization
**Learning**: Combine queries where possible. The hybrid search reuses results instead of running everything twice.

## Development Workflow

### 1. Logging is Your Best Friend
**Learning**: Added panic recovery and detailed logging after mysterious crashes. Always log entry IDs for traceability.

### 2. Hot Reload Complexity
**Learning**: Vite's HMR is amazing but can mask certain issues. Sometimes a full restart reveals problems.

### 3. Make Targets Save Time
**Learning**: Common operations should be one command away. `make dev`, `make test`, `make tail-log` improve developer experience significantly.

## Future Considerations

### 1. Scaling Embeddings
With 768-dimensional vectors, each entry uses ~3KB for embeddings alone. Plan for storage growth.

### 2. Model Selection
Qwen 2.5:7b is great for general analysis, but specialized models might better serve specific use cases (mental health, technical notes, creative writing).

### 3. Privacy First
All processing happens locally, but future cloud sync features must maintain end-to-end encryption.

### 4. Batch Processing
Current design processes one entry at a time. Batch processing could improve efficiency for bulk imports.

## What Would I Do Differently?

1. **Start with OpenTelemetry**: Better observability from day one would have made debugging easier.

2. **Database Migrations Tool**: Would use a proper migration tool like golang-migrate instead of raw SQL.

3. **Component Library**: Building a small component library first would have ensured more consistent UI.

4. **Integration Tests**: Would write integration tests for the full AI pipeline early to catch timeout issues.

5. **Configuration Management**: Would use Viper or similar for better config management instead of just environment variables.

## Key Takeaways

1. **AI enrichment should always be asynchronous**
2. **Test with real models early - mock responses hide issues**
3. **Database setup is often the hardest part for users**
4. **Clear separation of concerns enables future extensibility**
5. **User feedback must be immediate, even if processing continues**
6. **Logging and error handling are not optional**
7. **Version compatibility (especially with extensions) requires careful attention**
8. **The "dumbest thing that works" is often the most maintainable**

## Final Thoughts

Building an AI-powered application requires thinking differently about traditional patterns. The asynchronous nature of AI processing, the importance of embeddings for semantic search, and the need for extensible architectures all push you toward more modular, resilient designs. The combination of Go's simplicity, PostgreSQL's reliability, and local AI models creates a powerful foundation for privacy-preserving intelligent applications.