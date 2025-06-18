import axios from 'axios';

const API_URL = 'http://localhost:8080/api/rpc';

class JSONRPCClient {
  constructor() {
    this.id = 0;
  }

  async call(method, params) {
    const response = await axios.post(API_URL, {
      jsonrpc: '2.0',
      method,
      params,
      id: ++this.id,
    });

    if (response.data.error) {
      throw new Error(response.data.error.message);
    }

    return response.data.result;
  }
}

const client = new JSONRPCClient();

export const journalAPI = {
  // Journal entries
  createEntry: (content) => client.call('journal.create', { content }),
  updateEntry: (id, content) => client.call('journal.update', { id, content }),
  getEntry: (id) => client.call('journal.get', { id }),
  search: (params) => client.call('journal.search', params),
  toggleFavorite: (id) => client.call('journal.toggleFavorite', { id }),
  getProcessingLogs: (entryId) => client.call('journal.getProcessingLogs', { entry_id: entryId }),
  analyzeFailure: (entryId) => client.call('journal.analyzeFailure', { entry_id: entryId }),
  retryProcessing: (entryId) => client.call('journal.retryProcessing', { entry_id: entryId }),
  getSearchSuggestions: () => client.call('journal.getSearchSuggestions', {}),

  // Collections
  createCollection: (name, description) => 
    client.call('collection.create', { name, description }),
  getCollections: () => client.call('collection.list', {}),
  addToCollection: (entryId, collectionId) => 
    client.call('collection.addEntry', { entry_id: entryId, collection_id: collectionId }),
  removeFromCollection: (entryId, collectionId) => 
    client.call('collection.removeEntry', { entry_id: entryId, collection_id: collectionId }),

  // Evaluation endpoints
  generateTestData: (size = 100) => 
    client.call('evaluation.generateTestData', { size }),
  runEvaluation: (mode = 'all') => 
    client.call('evaluation.run', { mode }),
  generateReport: (format = 'html') => 
    client.call('evaluation.generateReport', { format }),
  getLatestResults: () => 
    client.call('evaluation.getLatestResults', {}),
  runFullEvaluation: (size = 100) => 
    client.call('evaluation.runFull', { size }),
};