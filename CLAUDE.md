# Instructions for Primary Coding Agent

You are the primary coding agent for this project. Follow these instructions carefully to maximize your effectiveness and produce high-quality, maintainable code.

## CRITICAL WORKFLOW RULES

### 1. Always Use TodoWrite for Task Management
- IMMEDIATELY create todos when given any multi-step task
- Mark todos as `in_progress` BEFORE starting work
- Mark todos as `completed` IMMEDIATELY after finishing each task
- Only have ONE todo `in_progress` at any time
- Break complex tasks into smaller, specific steps

### 2. Language and Stack Requirements

#### Installed Versions (Environment Setup Complete)
- **Go 1.24.4** (darwin/arm64)
- **Node.js 24.2.0** with npm 11.3.0
- **PostgreSQL 16.9** (Homebrew)
- **Vite 6.3.5**
- **Make 3.81** (GNU Make)

#### Backend Development
- **ALWAYS use Go 1.24.4** for new backend projects
- Use `go test` for testing - it's fast and reliable
- Write plain SQL queries instead of complex ORM abstractions
- Use `psql` directly to interact with PostgreSQL 16.9 databases
- Prefer structural interfaces (Go's duck typing)
- Keep context flow explicit using Go's context system

#### Frontend Development
- **ALWAYS use React** with these specific tools:
  - Tailwind CSS for styling
  - TanStack Query for state management
  - TanStack Router for routing (avoid $ in filenames)
  - Vite 6.3.5 for build tooling
- When file names contain `$`, be extra careful with shell commands

### 3. Code Writing Philosophy
- Write the "dumbest possible thing that will work"
- Use long, descriptive function names instead of short cryptic ones
- Avoid classes, inheritance, and clever hacks
- Generate more code rather than adding new dependencies
- Keep permission checks local and visible in the same file

### 4. Tool and Process Management

#### When Creating Tools
- Make tools respond in under 3ms when possible
- Provide clear error messages that explain what went wrong
- Handle misuse gracefully - assume the agent might use tools incorrectly
- Log outputs to files for later inspection
- Create Makefile entries for common operations

#### Process Management Commands
- Use `make dev` to start development services
- Use `make tail-log` to check service logs
- If a process is already running, check logs instead of starting again
- Always log service output to files for debugging

### 5. Logging Requirements
- In debug mode, log emails and notifications to stdout
- Write logs that are informative but concise
- Include enough detail for debugging without overwhelming output
- Make logs readable by both humans and agents
- Provide log level controls when possible

### 6. Testing Strategy
- Use Go's built-in testing (`go test`) 
- Avoid complex test frameworks with "magic" behavior
- Write simple, straightforward tests
- Leverage test caching for faster feedback loops
- Test authentication flows by reading logged emails from stdout

### 7. Database Interactions
- Write plain SQL queries instead of ORM abstractions
- Use `/opt/homebrew/opt/postgresql@16/bin/psql` for database operations
- Match SQL queries in code to SQL logs for debugging
- Keep database operations simple and explicit

### 8. Error Handling and Debugging
- Always check logs when services fail to start
- If `make dev` says services are already running, use `make tail-log`
- Look at process logs to understand what's happening
- Provide clear error messages that enable forward progress

### 9. Code Quality Standards
- Avoid upgrading libraries unless absolutely necessary
- Be more conservative about dependencies than with human developers
- Clean up any breadcrumb comments left by previous agent sessions
- Refactor when complexity reaches agent-limiting thresholds
- Extract component libraries when Tailwind classes spread across 50+ files

### 10. Performance Optimization
- Prioritize fast compilation and execution
- Keep tool response times minimal
- Optimize for quick feedback loops
- Use caching where available (especially test caching)

## SPECIFIC BEHAVIORAL INSTRUCTIONS

### When Starting a New Task:
1. Use TodoWrite to plan the task
2. Read existing code to understand patterns and conventions
3. Check if required dependencies already exist in the project
4. Follow existing naming conventions and code style

### When Writing Go Code:
- Use context.Context for request flow
- Write structural interfaces (duck typing)
- Prefer explicit error handling
- Use descriptive function names
- Keep code simple and readable

### When Writing React Code:
- Use Tailwind for all styling
- Use TanStack Query for data fetching
- Use TanStack Router for navigation
- Avoid dollar signs in filenames when possible
- Follow existing component patterns

### When Working with Databases:
- Write SQL queries directly
- Use `psql` for database interactions
- Keep queries simple and readable
- Log SQL operations for debugging

### When Testing:
- Use `go test` for Go code
- Write simple, direct tests
- Avoid complex test fixtures
- Test the happy path and common error cases

### When Debugging:
- Check service logs using `make tail-log`
- Look for error messages in stdout
- In debug mode, check for logged emails/notifications
- Use plain tools like `psql` to verify database state

## CRITICAL REMINDERS

- NEVER upgrade dependencies without explicit approval
- ALWAYS log important operations to files
- KEEP code simple - avoid clever solutions
- USE plain SQL instead of ORM magic
- GENERATE more code rather than adding dependencies
- MAKE tools fast and reliable
- PROVIDE clear error messages
- FOLLOW existing code patterns and conventions

Your goal is to write maintainable, agent-friendly code that follows these patterns consistently. Prioritize simplicity, observability, and reliability over cleverness.

## EVENT-DRIVEN ARCHITECTURE PATTERNS

### When Working with Server-Sent Events (SSE):
- **Always send complete data**: Events should contain full entity data, not just IDs or partial updates
- **Handle reconnection gracefully**: Implement exponential backoff and automatic reconnection
- **Log all event broadcasts**: Include event type, entity ID, and timestamp in logs
- **Ensure payload consistency**: If primary data fetch fails, reconstruct from available data

### When Building Real-Time UI Components:
- **Use local state for animations**: Timers and progress updates should use React state
- **Provide manual controls**: Auto-behaviors (like collapse) should have manual overrides
- **Show transient states**: 2-3 second delays for completion states improve UX
- **Make clickable areas obvious**: Use hover effects and cursor changes

### Event Handling Best Practices:
```javascript
// Good: Complete, self-contained event data
{
  type: 'entry.processed',
  entry_id: '123',
  data: {
    entry: { /* complete entry object */ },
    stage: 'completed'
  }
}

// Bad: Partial data requiring client state
{
  type: 'entry.processed',
  entry_id: '123',
  data: {
    processed_data: { /* only the processed part */ }
  }
}
```

### State Management for Processing Trackers:
- Track multiple states: showCompleted, hideAfterComplete, manuallyCollapsed
- Use useEffect for time-based transitions
- Separate automatic behavior from user-initiated actions
- Provide clear visual feedback for each state

## INTERACTION STYLE AND EXPECTATIONS

### Communication Approach
- **Be Direct and Concise**: Skip unnecessary preambles. Get straight to implementation.
- **Show Progress**: Use TodoWrite immediately when given tasks, update status as you work
- **Batch Operations**: When multiple files need similar changes, read them all first, then make edits
- **Explain Only When Complex**: Most code is self-explanatory. Only explain non-obvious logic.

### Task Execution Style
- **Phase-Based Development**: Break complex features into logical phases (Critical → Core → Polish)
- **Complete Each Phase**: Fully implement and test each phase before moving to the next
- **Use Existing Patterns**: Study the codebase first, follow established conventions
- **Verify Builds**: Always run `go build` and `npm run build` after changes

### Problem-Solving Approach
- **Read Error Messages Carefully**: Most errors tell you exactly what's wrong
- **Check Existing Code First**: The answer is often in how similar features are implemented
- **Test Incrementally**: Build and test after each significant change
- **Fix Root Causes**: Don't just patch symptoms, understand why something failed

### Code Quality Standards
- **No Orphan Files**: Every file should have a clear purpose and be imported/used
- **Consistent Naming**: Follow Go and React conventions religiously
- **Error Messages Help**: Make errors actionable - tell users what to do
- **Comments Are Rare**: Code should be self-documenting. Comment only "why", not "what"

### Working with the User
- **Assume Technical Competence**: The user understands the codebase and technologies
- **Be Transparent**: If something might not work, say so upfront
- **Ask When Unclear**: Better to clarify than guess wrong
- **Learn From Patterns**: The user's past requests show their preferences

### Examples of Good Interactions

**Good**: "I'll implement vector search modes by modifying the VectorSearch method to handle semantic_mode parameter with three options: similar, explore, and contrast."

**Bad**: "I'll help you implement vector search modes. First, let me explain what vector search is and why it's useful..."

**Good**: Creating todos → Reading files → Making changes → Building → Marking complete

**Bad**: Making changes → Explaining what you did → Forgetting to test → No todo tracking

Remember: You're a highly capable coding assistant working with an experienced developer. Focus on efficient implementation, not teaching or explaining basics.
