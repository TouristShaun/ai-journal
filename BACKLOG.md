# AI-Powered Journal App - Feature Backlog

## Priority Legend
- 🔴 **Critical**: Core functionality, bugs, or security issues
- 🟡 **High**: Important features that significantly improve UX
- 🟢 **Medium**: Nice-to-have features that enhance the app
- 🔵 **Low**: Future considerations and optimizations

---

## 🔴 Critical Priority

### Bug Fixes
- [x] Fix PostgreSQL search suggestions query error: "for SELECT DISTINCT, ORDER BY expressions must appear in select list" *(Fixed with LATERAL joins)*
- [ ] Handle Ollama model timeout gracefully (currently hardcoded 120s may not be enough for very large entries)
- [ ] Fix potential race condition in SSE event broadcasting when multiple clients connect simultaneously

---

## 🟡 High Priority

### Search/Filter Evaluation System ✅
- [x] **Test Data Generator**
  - [x] Generate synthetic journal entries with known characteristics
  - [x] Create entries with specific keywords, topics, entities, sentiments
  - [x] Build test sets for different search scenarios
  - [x] Include entries with varying similarity relationships

- [x] **Evaluation Framework**
  - [x] Core evaluation engine to run test cases
  - [x] Metrics collection (Precision, Recall, F1, NDCG, MRR)
  - [x] Performance measurement (latency, memory, throughput)
  - [x] Result comparison and scoring

- [x] **Test Case Implementation**
  - [x] Classic search test suite (keywords, filters, edge cases)
  - [x] Vector search test suite (similarity, modes, embedding quality)
  - [x] Hybrid search test suite (strategy effectiveness, score validation)
  - [x] User experience metrics (diversity, filter effectiveness)

- [x] **Reporting System**
  - [x] HTML report generator with visualizations
  - [x] JSON/CSV export for metrics
  - [x] Baseline comparison tracking
  - [ ] A/B testing framework *(foundation created, full implementation pending)*

### Performance Optimizations
- [ ] Implement connection pooling for PostgreSQL
- [ ] Add caching layer for frequently accessed entries
- [ ] Optimize embedding generation batch processing
- [ ] Implement lazy loading for large entry lists
- [ ] Add pagination to vector search results

### "Claude Code as an MCP Server" Integration & Dynamic Module System
**NOTE:** Claude Code as an MCP Server simply exposes Claude Code’s tools to your MCP client, so your own client is responsible for implementing user confirmation for individual tool calls. See @schema.ts for latest specification and @mcp-hosts.md for best practices when designing a host. The server provides access to Claude’s tools like View, Edit, LS, etc. Add Claude Code MCP server via:
    """
    {
        "command": "claude",
        "args": ["mcp", "serve"],
        "env": {}
    }
    """

- [ ] **MCP Server Integration**
  - [ ] Connect to Claude Code as MCP server using command: `claude mcp serve`
  - [ ] Create MCP client wrapper for secure communication
  - [ ] Expose Claude's tools (View, Edit, LS, etc.) through journal interface
  - [ ] Implement authentication and session management for MCP connection

- [ ] **Module/Subapp Framework**
  - [ ] Design module architecture with isolated namespaces
  - [ ] Create module registry and lifecycle management
  - [ ] Implement module scaffolding templates
  - [ ] Build inter-module communication system
  - [ ] Enforce shared techstack (Go, React, PostgreSQL) with extensibility hooks

- [ ] **Playground Environment**
  - [ ] Create isolated database schema for module development
  - [ ] Implement sandboxed execution environment
  - [ ] Build module testing and debugging tools
  - [ ] Add rollback and versioning for module changes
  - [ ] Create module backup and restore functionality

- [ ] **Journal-Driven Development**
  - [ ] Parse journal entries for module specifications
  - [ ] Generate module boilerplate from natural language descriptions
  - [ ] Track module development history in journal entries
  - [ ] Link modules to their originating journal entries
  - [ ] Enable collaborative module development through shared entries

- [ ] **Module UI Components**
  - [ ] Create module launcher interface
  - [ ] Build module configuration panels
  - [ ] Implement module preview system
  - [ ] Add module marketplace/gallery view
  - [ ] Design module permission management UI

- [ ] **Technical Requirements**
  - [ ] Separate module database tables with `module_` prefix
  - [ ] Module-specific API endpoints under `/api/modules/:moduleId`
  - [ ] React component lazy loading for modules
  - [ ] Module-specific state management
  - [ ] WebSocket support for real-time module updates

### Monetization & Pricing Model
- [ ] **Pricing Tiers Implementation**
  - [ ] Basic Journal App: $25 one-time lifetime license
  - [ ] Solopreneur Package: $99/month or $999/year (includes Claude Code MCP integration w/ Journal-Driven Modules Development)
  - [ ] License key generation and validation system
  - [ ] Stripe integration for payments
  - [ ] Subscription management portal

- [ ] **License Management**
  - [ ] License key generation algorithm
  - [ ] Offline license validation

### Authentication & Account System
- [ ] **Magic Link Authentication**
  - [ ] Email-based passwordless login
  - [ ] Magic link generation and expiration
  - [ ] Rate limiting for magic link requests
  - [ ] Email template customization
  - [ ] Domain whitelist for business accounts

- [ ] **Passkey Implementation**
  - [ ] WebAuthn integration for passkeys
  - [ ] Passkey registration flow after purchase
  - [ ] Multiple passkey support per account
  - [ ] Passkey recovery via email fallback
  - [ ] Cross-device passkey sync

### Electron Desktop Application
- [ ] **Core Electron Setup**
  - [ ] Electron app scaffolding
  - [ ] Auto-updater implementation
  - [ ] Code signing for Mac/Windows
  - [ ] Native menu integration
  - [ ] System tray support

- [ ] **PGlite Integration**
  - [ ] PGlite setup for local PostgreSQL
  - [ ] pgvector extension for PGlite
  - [ ] Local data migration tools
  - [ ] Performance optimization for local queries
  - [ ] Data integrity checks

- [ ] **Offline-First Architecture**
  - [ ] Complete offline functionality
  - [ ] Local Ollama integration
  - [ ] Queue system for sync operations
  - [ ] Conflict resolution strategies
  - [ ] Local backup system

- [ ] **Cloud Sync (Optional)**
  - [ ] Opt-in cloud backup toggle
  - [ ] End-to-end encryption for cloud data
  - [ ] Incremental sync algorithm
  - [ ] Bandwidth optimization
  - [ ] Sync status indicators

### User Profile & Settings
- [ ] **Profile Management**
  - [ ] User avatar upload
  - [ ] Display name and bio
  - [ ] Timezone settings
  - [ ] Language preferences
  - [ ] Account creation date display

- [ ] **Application Settings**
  - [ ] Theme customization (dark/light/auto)
  - [ ] Font family and size options
  - [ ] Editor preferences
  - [ ] Keyboard shortcut customization
  - [ ] AI model preferences

- [ ] **Data Management**
  - [ ] Export all data as archive
  - [ ] Import data from archive
  - [ ] Storage usage visualization
  - [ ] Data retention policies
  - [ ] GDPR compliance tools

### User Analytics Dashboard
- [ ] **Writing Analytics**
  - [ ] Daily/weekly/monthly word counts
  - [ ] Writing streak tracking
  - [ ] Most productive times
  - [ ] Entry frequency graphs
  - [ ] Average entry length trends

- [ ] **AI Insights**
  - [ ] Topic frequency analysis
  - [ ] Sentiment trends over time
  - [ ] Entity mention tracking
  - [ ] Mood correlation patterns
  - [ ] Personal growth indicators

- [ ] **Usage Metrics**
  - [ ] Feature usage statistics
  - [ ] Search pattern analysis
  - [ ] Collection usage trends
  - [ ] Processing time analytics
  - [ ] Error rate monitoring

---

## 🟢 Medium Priority

### Enhanced AI Features
- [ ] **Multi-model Support**
  - [ ] Allow users to choose between different Ollama models
  - [ ] Support for specialized models (coding, medical, creative writing)
  - [ ] Model performance comparison tools
  - [ ] Automatic model selection based on content type

- [ ] **Advanced Processing**
  - [ ] Extract action items and todos from entries
  - [ ] Identify recurring themes across time periods
  - [ ] Generate weekly/monthly summaries
  - [ ] Mood tracking and visualization

### Collaboration Features
- [ ] Shared collections with permission management
- [ ] Comments and annotations on entries
- [ ] Entry templates for consistent formatting
- [ ] Public/private entry toggles

---

## 🔵 Low Priority

### Advanced Analytics
- [ ] Sentiment trends over time
- [ ] Word frequency analysis
- [ ] Reading time estimates
- [ ] Entry complexity scoring
- [ ] Network graph of topic relationships

### UI/UX Enhancements
- [ ] Customizable themes and color schemes
- [ ] Font selection and size preferences
- [ ] Distraction-free writing mode
- [ ] Split-screen entry comparison
- [ ] Markdown preview toggle

---

## Completed Features ✅

### Phase 1-3 Implementation
- [x] Basic journal CRUD operations
- [x] AI-powered content analysis (Qwen)
- [x] Semantic search with embeddings
- [x] Three search modes (Classic, Vector, Hybrid)
- [x] Real-time updates with SSE
- [x] Processing tracker with visual feedback
- [x] Keyboard shortcuts
- [x] Export functionality (JSON, Markdown, CSV)
- [x] Retry failed processing
- [x] Search suggestions
- [x] Enhanced error handling

### Phase 4 Implementation (Latest)
- [x] Fixed PostgreSQL search suggestions query with LATERAL joins
- [x] Implemented complete Search/Filter Evaluation System
  - Test data generator with synthetic entries
  - Evaluation framework with comprehensive metrics
  - Test cases for all search modes
  - HTML/JSON/CSV reporting system
  - Makefile commands for easy execution
- [x] Added topbar with key features
  - Journal title and branding
  - Collections management modal
  - Evaluations UI (demo mode)
  - Keyboard shortcuts access
  - Collapsible sidebar toggle
- [x] Enhanced UI/UX
  - Collapsible sidebar with smooth transitions
  - Better space utilization
  - Improved navigation and accessibility
- [x] **Evaluation System UI Integration** (Phase 5 - Latest)
  - Fully functional evaluation UI in React frontend
  - Real-time progress tracking via SSE events
  - Display of all metrics (Precision, Recall, F1, NDCG, MRR)
  - Report generation (HTML, CSV, JSON) from UI
  - Automatic detection of existing evaluation results

---

## Notes

### Prioritization Criteria
1. **User Impact**: How many users will benefit?
2. **Technical Complexity**: How difficult is implementation?
3. **Dependencies**: What needs to be built first?
4. **Resource Requirements**: Time, compute, storage needs
5. **Strategic Value**: Does it differentiate the product?

### Next Sprint Recommendations
1. ~~Fix the PostgreSQL search query bug (Critical)~~ ✅
2. ~~Implement the evaluation system (High)~~ ✅ *Full UI integration completed*
3. Add batch import functionality (High)
4. Improve performance with caching (High)

### Long-term Vision
The journal app should evolve into a comprehensive personal knowledge management system that:
- Learns from user patterns
- Provides actionable insights
- Integrates seamlessly with daily workflows
- Maintains absolute privacy
- Scales to lifetime usage

---

*Last Updated: June 16, 2025*
