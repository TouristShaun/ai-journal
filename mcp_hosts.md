# MCP Host/Client Development Guide

You are an expert MCP (Model Context Protocol) host/client developer. This guide synthesizes all knowledge from the documentation in this folder to help you build robust, secure, and user-friendly MCP clients and host applications.

## Core Understanding

### What is an MCP Host/Client?
- **Purpose**: Applications that connect to MCP servers to extend LLM capabilities
- **Architecture**: Hosts manage multiple MCP client connections to different servers
- **Communication**: JSON-RPC 2.0 protocol for standardized messaging
- **Design Philosophy**: Hosts aggregate capabilities from multiple servers into unified experiences

### Key Principles
1. **User Control**: Always prioritize user security and approval
2. **Graceful Degradation**: Handle server unavailability elegantly
3. **Performance**: Maintain responsive UI even with multiple connections
4. **Security First**: Validate all server responses and implement sandboxing

## Implementation Strategy

### 1. Client Architecture

```typescript
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';

class MCPHost {
  private clients: Map<string, Client> = new Map();
  private capabilities: AggregatedCapabilities;
  
  async connectToServer(config: ServerConfig): Promise<void> {
    const client = new Client({
      name: 'my-mcp-host',
      version: '1.0.0',
    }, {
      capabilities: {}
    });
    
    // Choose transport based on server type
    const transport = this.createTransport(config);
    await client.connect(transport);
    
    // Store client with namespace
    this.clients.set(config.name, client);
    
    // Update aggregated capabilities
    await this.updateCapabilities();
  }
  
  private createTransport(config: ServerConfig): Transport {
    if (config.transport === 'stdio') {
      return new StdioClientTransport({
        command: config.command,
        args: config.args,
        env: config.env
      });
    } else if (config.transport === 'http') {
      return new HttpClientTransport({
        url: config.url,
        headers: config.headers
      });
    }
  }
}
```

### 2. Connection Management

```typescript
class ConnectionManager {
  private reconnectAttempts = new Map<string, number>();
  private healthChecks = new Map<string, NodeJS.Timer>();
  
  async manageConnection(
    serverName: string, 
    client: Client
  ): Promise<void> {
    // Monitor connection health
    this.healthChecks.set(serverName, setInterval(async () => {
      try {
        await client.ping();
      } catch (error) {
        await this.handleDisconnection(serverName, client);
      }
    }, 30000));
    
    // Handle disconnections
    client.on('disconnect', () => {
      this.handleDisconnection(serverName, client);
    });
  }
  
  private async handleDisconnection(
    serverName: string, 
    client: Client
  ): Promise<void> {
    const attempts = this.reconnectAttempts.get(serverName) || 0;
    
    if (attempts < 3) {
      // Exponential backoff
      const delay = Math.pow(2, attempts) * 1000;
      setTimeout(() => {
        this.reconnect(serverName, client);
      }, delay);
      
      this.reconnectAttempts.set(serverName, attempts + 1);
    } else {
      // Notify user and disable server
      this.notifyUserOfFailure(serverName);
    }
  }
}
```

### 3. Capability Aggregation

```typescript
interface AggregatedCapabilities {
  resources: Map<string, ResourceWithServer>;
  tools: Map<string, ToolWithServer>;
  prompts: Map<string, PromptWithServer>;
}

class CapabilityAggregator {
  aggregate(clients: Map<string, Client>): AggregatedCapabilities {
    const capabilities: AggregatedCapabilities = {
      resources: new Map(),
      tools: new Map(),
      prompts: new Map()
    };
    
    for (const [serverName, client] of clients) {
      // Aggregate resources
      const resources = client.getResources();
      for (const resource of resources) {
        const key = `${serverName}/${resource.uri}`;
        capabilities.resources.set(key, {
          ...resource,
          server: serverName,
          client
        });
      }
      
      // Aggregate tools with namespace
      const tools = client.getTools();
      for (const tool of tools) {
        const key = `${serverName}/${tool.name}`;
        capabilities.tools.set(key, {
          ...tool,
          server: serverName,
          client
        });
      }
      
      // Similar for prompts...
    }
    
    return capabilities;
  }
}
```

### 4. Resource Management

#### Discovery and Browsing
```typescript
class ResourceBrowser extends React.Component {
  state = {
    resources: [],
    selectedResource: null,
    searchQuery: '',
    filterByServer: null
  };
  
  async componentDidMount() {
    await this.loadResources();
  }
  
  async loadResources() {
    const resources = await this.props.host.listAllResources();
    this.setState({ resources });
  }
  
  render() {
    const filtered = this.filterResources();
    
    return (
      <div className="resource-browser">
        <SearchBar 
          value={this.state.searchQuery}
          onChange={this.handleSearch}
        />
        
        <ServerFilter
          servers={this.props.servers}
          selected={this.state.filterByServer}
          onChange={this.handleFilterChange}
        />
        
        <ResourceList
          resources={filtered}
          onSelect={this.handleResourceSelect}
        />
        
        {this.state.selectedResource && (
          <ResourceViewer
            resource={this.state.selectedResource}
            onSubscribe={this.handleSubscribe}
          />
        )}
      </div>
    );
  }
}
```

#### Caching Strategy
```typescript
class ResourceCache {
  private cache = new Map<string, CachedResource>();
  private maxSize = 100 * 1024 * 1024; // 100MB
  private currentSize = 0;
  
  async get(uri: string, fetcher: () => Promise<Resource>): Promise<Resource> {
    const cached = this.cache.get(uri);
    
    if (cached && !this.isStale(cached)) {
      // Move to front (LRU)
      this.cache.delete(uri);
      this.cache.set(uri, cached);
      return cached.resource;
    }
    
    // Fetch and cache
    const resource = await fetcher();
    this.set(uri, resource);
    return resource;
  }
  
  private set(uri: string, resource: Resource): void {
    const size = this.calculateSize(resource);
    
    // Evict if necessary
    while (this.currentSize + size > this.maxSize && this.cache.size > 0) {
      const [oldestKey] = this.cache.keys();
      this.evict(oldestKey);
    }
    
    this.cache.set(uri, {
      resource,
      timestamp: Date.now(),
      size
    });
    
    this.currentSize += size;
  }
}
```

### 5. Tool Invocation

#### Human Approval System
```typescript
interface ToolApprovalRequest {
  tool: Tool;
  arguments: any;
  context: string;
  riskLevel: 'low' | 'medium' | 'high';
}

class ToolApprovalManager {
  private approvalMode: 'manual' | 'auto' | 'trusted' = 'manual';
  private trustedTools = new Set<string>();
  
  async requestApproval(
    request: ToolApprovalRequest
  ): Promise<boolean> {
    // Check if tool requires approval
    if (this.approvalMode === 'auto') {
      return true;
    }
    
    if (this.approvalMode === 'trusted' && 
        this.trustedTools.has(request.tool.name)) {
      return true;
    }
    
    // Show approval UI
    return await this.showApprovalDialog(request);
  }
  
  private async showApprovalDialog(
    request: ToolApprovalRequest
  ): Promise<boolean> {
    return new Promise((resolve) => {
      const dialog = new ApprovalDialog({
        title: `Approve Tool: ${request.tool.name}`,
        description: request.tool.description,
        arguments: request.arguments,
        riskLevel: request.riskLevel,
        onApprove: () => {
          if (dialog.trustFutureUses) {
            this.trustedTools.add(request.tool.name);
          }
          resolve(true);
        },
        onDeny: () => resolve(false)
      });
      
      dialog.show();
    });
  }
}
```

#### Tool Execution with Sandboxing
```typescript
class ToolExecutor {
  private sandboxes = new Map<string, Sandbox>();
  
  async executeTool(
    tool: ToolWithServer,
    args: any,
    options: ExecutionOptions = {}
  ): Promise<ToolResult> {
    // Get approval if needed
    if (!await this.approvalManager.requestApproval({
      tool,
      arguments: args,
      context: options.context,
      riskLevel: this.assessRisk(tool, args)
    })) {
      throw new Error('Tool execution denied by user');
    }
    
    // Execute in sandbox if required
    if (tool.requiresSandbox) {
      return await this.executeInSandbox(tool, args);
    }
    
    // Direct execution with timeout
    const timeout = options.timeout || 30000;
    return await this.executeWithTimeout(
      tool.client.executeTool(tool.name, args),
      timeout
    );
  }
  
  private async executeInSandbox(
    tool: ToolWithServer,
    args: any
  ): Promise<ToolResult> {
    const sandbox = this.getSandbox(tool.server);
    
    return await sandbox.execute(async () => {
      return await tool.client.executeTool(tool.name, args);
    });
  }
}
```

### 6. Prompt Management

```typescript
class PromptManager {
  private prompts = new Map<string, PromptWithServer>();
  private suggestionEngine: PromptSuggestionEngine;
  
  async discoverPrompts(): Promise<void> {
    for (const [server, client] of this.clients) {
      const prompts = await client.listPrompts();
      
      for (const prompt of prompts) {
        this.prompts.set(`${server}/${prompt.name}`, {
          ...prompt,
          server,
          client
        });
      }
    }
    
    // Update suggestion engine
    this.suggestionEngine.updatePrompts(this.prompts);
  }
  
  async executePrompt(
    promptKey: string,
    arguments: Record<string, any>
  ): Promise<PromptResult> {
    const prompt = this.prompts.get(promptKey);
    if (!prompt) {
      throw new Error(`Prompt not found: ${promptKey}`);
    }
    
    // Validate arguments
    const validation = this.validateArguments(
      prompt.arguments,
      arguments
    );
    
    if (!validation.valid) {
      throw new Error(`Invalid arguments: ${validation.errors.join(', ')}`);
    }
    
    // Get prompt messages
    const messages = await prompt.client.getPrompt(
      prompt.name,
      arguments
    );
    
    // Execute with LLM
    return await this.llm.complete(messages);
  }
}

// Dynamic form generation
class PromptForm extends React.Component {
  renderArgumentField(arg: PromptArgument) {
    const { name, description, required } = arg;
    
    return (
      <div key={name} className="form-field">
        <label>
          {name} {required && <span className="required">*</span>}
        </label>
        <input
          type="text"
          placeholder={description}
          value={this.state.values[name] || ''}
          onChange={(e) => this.updateValue(name, e.target.value)}
          required={required}
        />
      </div>
    );
  }
  
  render() {
    const { prompt } = this.props;
    
    return (
      <form onSubmit={this.handleSubmit}>
        <h3>{prompt.name}</h3>
        <p>{prompt.description}</p>
        
        {prompt.arguments.map(arg => 
          this.renderArgumentField(arg)
        )}
        
        <button type="submit">Execute Prompt</button>
      </form>
    );
  }
}
```

### 7. Sampling Security

```typescript
class SamplingSecurityManager {
  private permissions: SamplingPermissions = {
    mode: 'manual', // manual, auto, restricted
    maxTokens: 1000,
    allowedServers: new Set(),
    tokenBudget: {
      daily: 100000,
      perRequest: 5000
    }
  };
  
  async handleSamplingRequest(
    request: SamplingRequest,
    server: string
  ): Promise<SamplingResponse> {
    // Check permissions
    const allowed = await this.checkPermissions(request, server);
    if (!allowed) {
      throw new Error('Sampling not permitted');
    }
    
    // Apply filters
    const filtered = await this.applyFilters(request);
    
    // Check token budget
    if (!this.checkTokenBudget(filtered)) {
      throw new Error('Token budget exceeded');
    }
    
    // Get user approval if needed
    if (this.permissions.mode === 'manual') {
      const approved = await this.getUserApproval(filtered);
      if (!approved) {
        throw new Error('User denied sampling request');
      }
    }
    
    // Execute sampling
    const response = await this.llm.sample(filtered);
    
    // Update budgets
    this.updateTokenUsage(response.usage);
    
    return response;
  }
  
  private async applyFilters(
    request: SamplingRequest
  ): Promise<SamplingRequest> {
    // Filter system prompts
    if (request.systemPrompt && this.filters.systemPrompt) {
      request.systemPrompt = await this.filters.systemPrompt(
        request.systemPrompt
      );
    }
    
    // Filter messages
    if (request.messages && this.filters.messages) {
      request.messages = await this.filters.messages(
        request.messages
      );
    }
    
    // Apply token limits
    request.maxTokens = Math.min(
      request.maxTokens || this.permissions.maxTokens,
      this.permissions.tokenBudget.perRequest
    );
    
    return request;
  }
}
```

### 8. Root Management

```typescript
class RootManager {
  private roots = new Map<string, Root[]>();
  
  async updateRoots(server: string, roots: Root[]): Promise<void> {
    // Validate roots
    for (const root of roots) {
      if (!this.isValidRoot(root)) {
        throw new Error(`Invalid root: ${root.uri}`);
      }
    }
    
    // Update stored roots
    this.roots.set(server, roots);
    
    // Notify servers of changes
    const client = this.clients.get(server);
    if (client) {
      await client.notify('roots/list_changed', { roots });
    }
    
    // Update UI
    this.emit('roots:changed', { server, roots });
  }
  
  getRootsForServer(server: string): Root[] {
    return this.roots.get(server) || [];
  }
  
  canAccessResource(
    server: string, 
    resourceUri: string
  ): boolean {
    const roots = this.getRootsForServer(server);
    
    if (roots.length === 0) {
      // No roots means unrestricted
      return true;
    }
    
    return roots.some(root => 
      this.isUriWithinRoot(resourceUri, root)
    );
  }
}

// Root picker UI
class RootPicker extends React.Component {
  render() {
    return (
      <div className="root-picker">
        <h3>Select Working Directories</h3>
        
        {this.state.roots.map((root, index) => (
          <div key={index} className="root-item">
            <input
              type="text"
              value={root.uri}
              onChange={(e) => this.updateRoot(index, e.target.value)}
              placeholder="file:///path/to/directory"
            />
            <button onClick={() => this.browseRoot(index)}>
              Browse
            </button>
            <button onClick={() => this.removeRoot(index)}>
              Remove
            </button>
          </div>
        ))}
        
        <button onClick={this.addRoot}>Add Root</button>
        <button onClick={this.saveRoots}>Save</button>
      </div>
    );
  }
}
```

### 9. Transport Selection

```typescript
class TransportSelector {
  selectTransport(serverConfig: ServerConfig): TransportType {
    // Decision tree for transport selection
    if (serverConfig.type === 'local') {
      // Local tools/scripts
      return 'stdio';
    }
    
    if (serverConfig.type === 'remote') {
      // Remote services
      return 'http';
    }
    
    if (serverConfig.requiresAuth) {
      // Services needing authentication
      return 'http'; // Better auth support
    }
    
    if (serverConfig.performance === 'critical') {
      // Performance-critical
      return 'stdio'; // Lower overhead
    }
    
    // Default
    return 'stdio';
  }
  
  createTransportConfig(
    type: TransportType,
    config: ServerConfig
  ): TransportConfig {
    switch (type) {
      case 'stdio':
        return {
          command: config.command,
          args: config.args || [],
          env: {
            ...process.env,
            ...config.env
          },
          cwd: config.cwd
        };
        
      case 'http':
        return {
          url: config.url,
          headers: {
            'Authorization': config.auth?.token 
              ? `Bearer ${config.auth.token}` 
              : undefined,
            ...config.headers
          },
          timeout: config.timeout || 30000
        };
    }
  }
}
```

### 10. Debugging Infrastructure

```typescript
class DebugManager {
  private devTools: DevTools;
  private messageLog: MessageLog;
  private performanceMonitor: PerformanceMonitor;
  
  enableDebugging(): void {
    // Enable DevTools
    if (process.env.NODE_ENV === 'development') {
      this.devTools.enable();
    }
    
    // Start logging
    this.startMessageLogging();
    
    // Monitor performance
    this.performanceMonitor.start();
  }
  
  private startMessageLogging(): void {
    for (const [server, client] of this.clients) {
      client.on('message:sent', (msg) => {
        this.messageLog.log({
          direction: 'sent',
          server,
          message: msg,
          timestamp: Date.now()
        });
      });
      
      client.on('message:received', (msg) => {
        this.messageLog.log({
          direction: 'received',
          server,
          message: msg,
          timestamp: Date.now()
        });
      });
    }
  }
  
  getDebugState(): DebugState {
    return {
      connections: this.getConnectionStates(),
      messageStats: this.messageLog.getStats(),
      performance: this.performanceMonitor.getMetrics(),
      errors: this.errorLog.getRecent()
    };
  }
}
```

## Best Practices Checklist

### Architecture
- [ ] Implement connection pooling and management
- [ ] Use proper capability aggregation with namespacing
- [ ] Build robust error handling and recovery
- [ ] Design for extensibility and future features

### Security
- [ ] Implement user approval workflows for tools
- [ ] Validate all server responses
- [ ] Apply sandboxing for untrusted operations
- [ ] Enforce token budgets for sampling
- [ ] Implement root-based access control
- [ ] Filter sensitive data from prompts

### Performance
- [ ] Cache resources intelligently with LRU eviction
- [ ] Implement request batching where possible
- [ ] Use appropriate transports for use cases
- [ ] Monitor connection health proactively
- [ ] Optimize UI updates with debouncing

### User Experience
- [ ] Provide clear feedback for all operations
- [ ] Show connection status prominently
- [ ] Make approval dialogs informative
- [ ] Handle errors gracefully
- [ ] Provide debugging tools in development

### Testing
- [ ] Test with multiple server connections
- [ ] Simulate connection failures
- [ ] Test approval workflows thoroughly
- [ ] Validate security boundaries
- [ ] Use MCP Inspector for protocol compliance

## Common Patterns

### Progressive Trust Model
```typescript
class TrustManager {
  private trustLevels = new Map<string, TrustLevel>();
  
  evaluateTrust(server: string): TrustLevel {
    const history = this.getServerHistory(server);
    
    if (history.deniedRequests > 0) {
      return 'untrusted';
    }
    
    if (history.approvedRequests < 10) {
      return 'new';
    }
    
    if (history.successRate > 0.95) {
      return 'trusted';
    }
    
    return 'standard';
  }
  
  getPermissions(trustLevel: TrustLevel): Permissions {
    switch (trustLevel) {
      case 'trusted':
        return {
          autoApprove: ['read', 'list'],
          requireApproval: ['write', 'delete'],
          maxTokens: 5000
        };
      
      case 'standard':
        return {
          autoApprove: ['list'],
          requireApproval: ['read', 'write', 'delete'],
          maxTokens: 1000
        };
      
      case 'new':
      case 'untrusted':
        return {
          autoApprove: [],
          requireApproval: ['all'],
          maxTokens: 500
        };
    }
  }
}
```

### Request Batching
```typescript
class RequestBatcher {
  private pending = new Map<string, PendingRequest[]>();
  private flushInterval = 50; // ms
  
  async request(
    server: string,
    method: string,
    params: any
  ): Promise<any> {
    return new Promise((resolve, reject) => {
      // Add to pending
      if (!this.pending.has(server)) {
        this.pending.set(server, []);
        
        // Schedule flush
        setTimeout(() => this.flush(server), this.flushInterval);
      }
      
      this.pending.get(server).push({
        method,
        params,
        resolve,
        reject
      });
    });
  }
  
  private async flush(server: string): Promise<void> {
    const requests = this.pending.get(server);
    if (!requests || requests.length === 0) return;
    
    this.pending.delete(server);
    
    try {
      // Send batch request
      const client = this.clients.get(server);
      const responses = await client.batch(
        requests.map(r => ({
          method: r.method,
          params: r.params
        }))
      );
      
      // Resolve individual promises
      responses.forEach((response, i) => {
        if (response.error) {
          requests[i].reject(response.error);
        } else {
          requests[i].resolve(response.result);
        }
      });
    } catch (error) {
      // Reject all
      requests.forEach(r => r.reject(error));
    }
  }
}
```

## Future Compatibility

### Prepare for Registry
```typescript
class RegistryClient {
  async discoverServers(query: DiscoveryQuery): Promise<ServerInfo[]> {
    const results = await this.registry.search(query);
    
    return results.map(server => ({
      name: server.name,
      description: server.description,
      capabilities: server.capabilities,
      transport: server.preferredTransport,
      installCommand: server.installCommand,
      rating: server.communityRating
    }));
  }
  
  async installServer(serverInfo: ServerInfo): Promise<void> {
    // Download and install
    await this.installer.install(serverInfo);
    
    // Verify installation
    const config = await this.verifyInstallation(serverInfo);
    
    // Add to host configuration
    await this.addServerConfig(config);
  }
}
```

### Agent Graph Support
```typescript
interface AgentGraph {
  nodes: AgentNode[];
  edges: AgentEdge[];
  entry: string;
}

class AgentGraphExecutor {
  async executeGraph(
    graph: AgentGraph,
    input: any
  ): Promise<any> {
    const execution = new GraphExecution(graph);
    
    // Start from entry node
    const entryNode = graph.nodes.find(n => n.id === graph.entry);
    
    return await this.executeNode(
      entryNode,
      input,
      execution
    );
  }
  
  private async executeNode(
    node: AgentNode,
    input: any,
    execution: GraphExecution
  ): Promise<any> {
    // Check permissions for node
    if (!await this.checkNodePermissions(node)) {
      throw new Error(`Permission denied for node: ${node.id}`);
    }
    
    // Execute node logic
    const result = await node.execute(input);
    
    // Record execution
    execution.recordNode(node.id, result);
    
    // Find next nodes
    const edges = execution.graph.edges.filter(
      e => e.from === node.id
    );
    
    // Execute next nodes based on conditions
    for (const edge of edges) {
      if (this.evaluateCondition(edge.condition, result)) {
        const nextNode = execution.graph.nodes.find(
          n => n.id === edge.to
        );
        
        return await this.executeNode(
          nextNode,
          result,
          execution
        );
      }
    }
    
    return result;
  }
}
```

## Remember

1. **User First**: Always prioritize user security and control
2. **Fail Gracefully**: Handle server unavailability without breaking the experience
3. **Clear Communication**: Show what's happening with server connections
4. **Performance Matters**: Keep the UI responsive even with many servers
5. **Security by Default**: Require approval for potentially dangerous operations
6. **Debug-Friendly**: Provide tools for understanding what's happening

## Documentation Reference Guide

When building your MCP host/client application, refer to these specialized documentation files for detailed guidance:

### Getting Started
- **[introduction.md](./introduction.md)** - MCP overview from the host/client perspective
  - When to read: Before starting any MCP client development
  - Key topics: Core concepts, host vs client distinction, basic setup

- **[architecture.md](./architecture.md)** - Client architecture patterns and connection management
  - When to read: When designing your host application structure
  - Key topics: Connection pooling, capability aggregation, state management

### Core Implementation
- **[transports.md](./transports.md)** - Transport selection and implementation for clients
  - When to read: When connecting to different types of servers
  - Key topics: Stdio vs HTTP/SSE, connection lifecycle, reconnection strategies

- **[resources.md](./resources.md)** - Resource discovery and consumption patterns
  - When to read: When implementing resource browsing and caching
  - Key topics: Resource UI patterns, caching strategies, subscription handling

- **[tools.md](./tools.md)** - Tool discovery, invocation, and safety from client perspective
  - When to read: When implementing tool execution with user approval
  - Key topics: Approval workflows, sandboxing, UI patterns for tools

- **[prompts.md](./prompts.md)** - Managing and executing prompt templates
  - When to read: When building prompt selection and execution UIs
  - Key topics: Dynamic forms, argument collection, prompt suggestions

### Security & Control
- **[sampling.md](./sampling.md)** - Managing server sampling requests securely
  - When to read: When implementing sampling permission systems
  - Key topics: Permission models, token budgets, content filtering

- **[roots.md](./roots.md)** - Root management for access control
  - When to read: When implementing file system boundaries
  - Key topics: Root UI patterns, permission boundaries, dynamic updates

### Development & Testing
- **[debugging.md](./debugging.md)** - Debugging tools and techniques for hosts
  - When to read: When troubleshooting connection or protocol issues
  - Key topics: DevTools usage, message logging, performance monitoring

- **[inspector.md](./inspector.md)** - Using MCP Inspector as reference implementation
  - When to read: When learning protocol behavior or testing servers
  - Key topics: Protocol learning, compliance testing, debugging workflows

### Integration Resources
- **[example_clients.md](./example_clients.md)** - Overview of existing MCP clients
  - When to read: When choosing a client or understanding the ecosystem
  - Key topics: Client types, selection criteria, integration patterns

- **[example_servers.md](./example_servers.md)** - Testing with example servers
  - When to read: When testing your client implementation
  - Key topics: Test scenarios, learning from examples, integration testing

### Contributing & Future
- **[contributing.md](./contributing.md)** - Guidelines for client/host contributions
  - When to read: When sharing your implementation with the community
  - Key topics: Code standards, security requirements, documentation

- **[roadmap.md](./roadmap.md)** - Upcoming features affecting hosts/clients
  - When to read: When planning for future compatibility
  - Key topics: Registry integration, agent graphs, multimodal support

- **[faqs.md](./faqs.md)** - Common questions for host implementers
  - When to read: When facing common implementation challenges
  - Key topics: Connection issues, performance tips, UI patterns

### Quick Reference by Task

**Setting up a new host?** Start with:
1. [introduction.md](./introduction.md) - Understand the basics
2. [architecture.md](./architecture.md) - Design your structure
3. [example_clients.md](./example_clients.md) - Learn from existing implementations

**Implementing server connections?** Refer to:
1. [transports.md](./transports.md) - Choose and implement transport
2. [debugging.md](./debugging.md) - Set up debugging early
3. [faqs.md](./faqs.md) - Avoid common pitfalls

**Building UI for capabilities?** Check:
1. [resources.md](./resources.md) - Resource browser patterns
2. [tools.md](./tools.md) - Tool approval interfaces
3. [prompts.md](./prompts.md) - Prompt selection UIs

**Ensuring security?** Study:
1. [sampling.md](./sampling.md) - Control LLM access
2. [roots.md](./roots.md) - Implement boundaries
3. [tools.md](./tools.md) - Tool sandboxing

By following this guide and referring to the specialized documentation as needed, you'll create MCP hosts that provide powerful, secure, and user-friendly experiences for extending LLM capabilities with external tools and data sources.

## JSON Schema Reference

The MCP protocol is formally defined in `schema.json`, which provides the complete specification for all message types, data structures, and protocol requirements. Understanding this schema is essential for building compliant MCP clients/hosts.

### Key Schema Components for Hosts/Clients

#### 1. Client Capabilities
Declare what your client supports during initialization:
```typescript
// From schema: ClientCapabilities
interface ClientCapabilities {
  roots?: {
    listChanged?: boolean;  // Support for roots list change notifications
  };
  sampling?: {};           // Support for LLM sampling
  experimental?: {         // Custom capabilities
    [key: string]: any;
  };
}

// Initialize with capabilities
const initRequest = {
  method: "initialize",
  params: {
    protocolVersion: "1.0",
    capabilities: {
      roots: { listChanged: true },
      sampling: {}
    },
    clientInfo: {
      name: "my-mcp-client",
      version: "1.0.0"
    }
  }
};
```

#### 2. Content Handling
Clients must handle various content types from servers:
```typescript
// When receiving tool results or prompts
type Content = TextContent | ImageContent | AudioContent | EmbeddedResource;

// Text handling (most common)
function handleTextContent(content: TextContent): void {
  const { text, annotations } = content;
  if (annotations?.audience?.includes("user")) {
    // Display to user
  }
}

// Image handling
function handleImageContent(content: ImageContent): void {
  const imageData = atob(content.data); // Decode base64
  // Display based on mimeType
}

// Embedded resource handling
function handleEmbeddedResource(content: EmbeddedResource): void {
  const { resource } = content;
  if ('text' in resource) {
    // Handle text resource
  } else {
    // Handle blob resource
  }
}
```

#### 3. Request/Response Pattern
All client requests follow JSON-RPC 2.0:
```typescript
// Request structure
interface JSONRPCRequest {
  jsonrpc: "2.0";
  id: string | number;     // Unique request ID
  method: string;
  params?: {
    _meta?: {
      progressToken?: string | number;  // For progress tracking
    };
    [key: string]: any;
  };
}

// Handle responses
function handleResponse(response: JSONRPCMessage): void {
  if ('error' in response) {
    // Handle error
    const { code, message, data } = response.error;
    switch (code) {
      case -32602: // Invalid params
      case -32601: // Method not found
      case -32603: // Internal error
        // Handle accordingly
    }
  } else if ('result' in response) {
    // Process successful result
    const { _meta, ...data } = response.result;
  }
}
```

#### 4. Root Management
Roots define operational boundaries:
```typescript
interface Root {
  uri: string;        // Must start with file:// currently
  name?: string;      // Human-readable identifier
}

// Respond to server requests for roots
async function handleListRootsRequest(): Promise<ListRootsResult> {
  const roots = await this.getUserSelectedRoots();
  return {
    roots: roots.map(r => ({
      uri: r.path.startsWith('file://') ? r.path : `file://${r.path}`,
      name: r.name
    }))
  };
}

// Notify servers of root changes
async function updateRoots(newRoots: Root[]): Promise<void> {
  this.roots = newRoots;
  
  // Notify all connected servers
  for (const [server, client] of this.clients) {
    await client.notify('notifications/roots/list_changed', {});
  }
}
```

#### 5. Sampling Requests
Handle server requests for LLM completions:
```typescript
interface SamplingMessage {
  role: "user" | "assistant";
  content: TextContent | ImageContent | AudioContent;
}

async function handleSamplingRequest(
  request: CreateMessageRequest
): Promise<CreateMessageResult> {
  const { messages, maxTokens, modelPreferences } = request.params;
  
  // Show approval UI to user
  const approved = await this.showSamplingApproval({
    messages,
    maxTokens,
    serverName: request.source
  });
  
  if (!approved) {
    throw new Error("User denied sampling request");
  }
  
  // Select model based on preferences
  const model = this.selectModel(modelPreferences);
  
  // Call LLM
  const response = await this.llm.complete({
    messages,
    maxTokens,
    model
  });
  
  return {
    role: "assistant",
    content: {
      type: "text",
      text: response.text
    },
    model: response.model,
    stopReason: response.stopReason
  };
}
```

#### 6. Tool Invocation
Validate and execute tools safely:
```typescript
interface Tool {
  name: string;
  description?: string;
  inputSchema: {
    type: "object";
    properties: Record<string, any>;
    required?: string[];
  };
  annotations?: ToolAnnotations;
}

async function invokeTool(
  server: string,
  toolName: string,
  args: any
): Promise<CallToolResult> {
  const tool = this.tools.get(`${server}/${toolName}`);
  
  // Validate arguments against schema
  const ajv = new Ajv();
  const validate = ajv.compile(tool.inputSchema);
  
  if (!validate(args)) {
    throw new Error(`Invalid arguments: ${ajv.errorsText(validate.errors)}`);
  }
  
  // Check tool annotations for safety
  if (!tool.annotations?.readOnlyHint) {
    // Show approval UI for potentially destructive operations
    const approved = await this.requestApproval({
      tool,
      args,
      destructive: tool.annotations?.destructiveHint ?? true
    });
    
    if (!approved) {
      throw new Error("Tool execution denied by user");
    }
  }
  
  // Execute tool
  const result = await this.clients.get(server).callTool({
    name: toolName,
    arguments: args
  });
  
  return result;
}
```

#### 7. Progress Tracking
Handle long-running operations:
```typescript
// Include progress token in requests
const request = {
  method: "resources/read",
  params: {
    uri: "large://file",
    _meta: {
      progressToken: generateToken()
    }
  }
};

// Handle progress notifications
client.on('notifications/progress', (notification) => {
  const { progressToken, progress, total, message } = notification.params;
  
  // Update UI
  this.updateProgress({
    token: progressToken,
    percent: total ? (progress / total) * 100 : undefined,
    message
  });
});
```

#### 8. Pagination
Handle large result sets:
```typescript
async function* getAllResources(server: string) {
  let cursor: string | undefined;
  
  do {
    const response = await this.clients.get(server).listResources({
      cursor
    });
    
    yield* response.resources;
    cursor = response.nextCursor;
  } while (cursor);
}

// Use with UI virtualization
const resources = [];
for await (const resource of getAllResources('myserver')) {
  resources.push(resource);
  if (resources.length % 100 === 0) {
    // Update UI with batch
    this.renderResources(resources);
  }
}
```

#### 9. Notifications
Handle server-initiated notifications:
```typescript
// Resource list changed
client.on('notifications/resources/list_changed', async () => {
  await this.refreshResourceList();
});

// Resource updated (subscribed)
client.on('notifications/resources/updated', (notification) => {
  const { uri } = notification.params;
  this.invalidateCache(uri);
  this.notifySubscribers(uri);
});

// Tool/prompt list changed
client.on('notifications/tools/list_changed', async () => {
  await this.refreshCapabilities();
});

// Log messages from server
client.on('notifications/message', (notification) => {
  const { level, data, logger } = notification.params;
  this.logConsole.append({
    level,
    message: data,
    source: logger,
    timestamp: Date.now()
  });
});
```

#### 10. Completion Support
Provide autocompletion for better UX:
```typescript
async function getCompletions(
  ref: PromptReference | ResourceReference,
  argument: { name: string; value: string }
): Promise<string[]> {
  const response = await client.complete({
    ref,
    argument
  });
  
  const { values, hasMore, total } = response.completion;
  
  if (hasMore && values.length < total) {
    // Could implement pagination or show indicator
    this.showMoreAvailable(total - values.length);
  }
  
  return values;
}
```

### Implementation Best Practices from Schema

1. **Protocol version negotiation**: Always check server's protocol version in InitializeResult
2. **Capability detection**: Only use features that both client and server support
3. **ID management**: Maintain unique request IDs for proper response correlation
4. **Notification handling**: Set up handlers before sending `initialized` notification
5. **Content type support**: Be prepared to handle all content types, even if just displaying
6. **Metadata preservation**: Pass through `_meta` fields in requests/responses
7. **Error recovery**: Implement proper error handling for all error codes
8. **URI validation**: Ensure all URIs follow proper format (especially roots)
9. **Schema validation**: Validate all data against schema definitions
10. **Graceful degradation**: Handle missing optional fields appropriately

### Schema-Driven Validation Example
```typescript
import Ajv from 'ajv';
import schema from './schema.json';

class SchemaValidator {
  private ajv = new Ajv();
  private validators = new Map();
  
  constructor() {
    // Pre-compile validators for common types
    this.validators.set('Tool', this.ajv.compile(schema.definitions.Tool));
    this.validators.set('Resource', this.ajv.compile(schema.definitions.Resource));
    this.validators.set('Prompt', this.ajv.compile(schema.definitions.Prompt));
  }
  
  validate(type: string, data: any): void {
    const validator = this.validators.get(type);
    if (!validator(data)) {
      throw new Error(
        `Invalid ${type}: ${this.ajv.errorsText(validator.errors)}`
      );
    }
  }
}
```

Understanding the JSON Schema helps ensure your client correctly handles all protocol features and edge cases, providing a robust foundation for MCP integration.

## Development Workflow Commands

This project includes Claude Code slash commands to streamline MCP host/client development. These commands provide a structured, three-step workflow for building any domain-specific MCP host application.

### Available Commands

#### 1. `/project:1-design-host [use-case]`
**Purpose**: Create a comprehensive design specification for your MCP host application

**Usage**: `/project:1-design-host research assistant` or `/project:1-design-host code analysis`

**What it creates**:
- Application purpose and user needs analysis
- Multi-server management architecture design
- User experience and interface planning
- Security and control framework specification
- Integration patterns with LLMs and external services
- Technical architecture including transport and concurrency handling

**Example**: For a "research assistant" use case, this command will design UI patterns for paper browsing, citation management workflows, and multi-source data aggregation.

#### 2. `/project:2-implement-host [use-case]`
**Purpose**: Implement the complete MCP host application based on the design from step 1

**Usage**: `/project:2-implement-host research assistant`

**What it creates**:
- Full React/TypeScript implementation using TanStack libraries
- Multi-server connection management with health monitoring
- Unified capability aggregation and presentation
- Secure tool execution with approval workflows
- Intelligent caching and state management
- Comprehensive UI components for the specific use case
- Error handling and recovery systems

**Example**: For a "research assistant" use case, this creates a working application with paper search interfaces, citation tools, and collaborative research features.

#### 3. `/project:3-deploy-host [use-case]`
**Purpose**: Create complete distribution package with multi-platform deployment

**Usage**: `/project:3-deploy-host research assistant`

**What it creates**:
- Multi-platform build pipeline (Web, Electron, VS Code extension)
- User-friendly configuration and setup wizards
- Comprehensive documentation and tutorials
- Automated testing and quality assurance setup
- Distribution infrastructure with auto-updates
- Integration guides for popular MCP servers
- Launch strategy and community support materials

**Example**: For a "research assistant" use case, this creates installable applications, browser extensions, and academic workflow integration guides.

### Workflow Example

Here's how to build a complete content management MCP host:

```bash
# Step 1: Design the host architecture
/project:1-design-host content management

# Step 2: Implement the complete application
/project:2-implement-host content management

# Step 3: Create distribution package
/project:3-deploy-host content management
```

Each command builds on the previous one, ensuring consistency and completeness throughout the development process.

### Command Benefits

- **User-Centric Design**: Every application is designed around specific user workflows and needs
- **Multi-Server Ready**: Built-in support for connecting to and managing multiple MCP servers
- **Security First**: Comprehensive approval workflows and permission management
- **Production Quality**: Includes testing, monitoring, and distribution infrastructure
- **Cross-Platform**: Supports web, desktop, and editor extension deployment
- **Best Practices**: Incorporates all the patterns and recommendations from this CLAUDE.md guide

### Use Case Examples

The commands work for any host application use case:

- **Development Tools**: `/project:1-design-host code review assistant`
- **Content Creation**: `/project:1-design-host writing workflow manager`
- **Data Analysis**: `/project:1-design-host research data explorer`
- **Project Management**: `/project:1-design-host team collaboration hub`
- **Education**: `/project:1-design-host learning assistant`
- **Healthcare**: `/project:1-design-host patient data coordinator`

### Integration Focus

Each generated host application includes:
- **Server Discovery**: Automatic detection and configuration of relevant MCP servers
- **Workflow Integration**: Seamless embedding into existing user workflows
- **Context Awareness**: Intelligent use of user context and preferences
- **Progressive Enhancement**: Graceful handling of server availability and capabilities

### Customization

After running the commands, you can customize the generated application to fit your specific requirements while maintaining the solid foundation of connection management, security controls, and user experience patterns provided by the structured workflow.