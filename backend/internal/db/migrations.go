package db

const CreateTablesSQL = `
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Collections table
CREATE TABLE IF NOT EXISTS collections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Journal entries table
CREATE TABLE IF NOT EXISTS journal_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    processed_data JSONB NOT NULL DEFAULT '{}',
    embedding vector(768), -- nomic-embed-text uses 768 dimensions
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_favorite BOOLEAN DEFAULT FALSE,
    original_entry_id UUID REFERENCES journal_entries(id),
    tsv tsvector GENERATED ALWAYS AS (to_tsvector('english', content)) STORED
);

-- Junction table for journal entries and collections
CREATE TABLE IF NOT EXISTS journal_collection (
    journal_id UUID REFERENCES journal_entries(id) ON DELETE CASCADE,
    collection_id UUID REFERENCES collections(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (journal_id, collection_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_journal_entries_created_at ON journal_entries(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_journal_entries_favorite ON journal_entries(is_favorite) WHERE is_favorite = TRUE;
CREATE INDEX IF NOT EXISTS idx_journal_entries_embedding ON journal_entries USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX IF NOT EXISTS idx_journal_entries_tsv ON journal_entries USING GIN(tsv);
CREATE INDEX IF NOT EXISTS idx_journal_entries_processed_data ON journal_entries USING GIN(processed_data);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers
CREATE TRIGGER update_collections_updated_at BEFORE UPDATE ON collections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_journal_entries_updated_at BEFORE UPDATE ON journal_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
`
