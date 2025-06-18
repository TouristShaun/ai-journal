package db

const AddProcessingTrackerSQL = `
-- Create enum type for processing stages
CREATE TYPE processing_stage AS ENUM (
    'created',
    'analyzing',
    'fetching_urls',
    'generating_embeddings',
    'completed',
    'failed'
);

-- Add processing stage columns to journal_entries
ALTER TABLE journal_entries 
ADD COLUMN IF NOT EXISTS processing_stage processing_stage DEFAULT 'created',
ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS processing_completed_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS processing_error TEXT;

-- Create processing logs table
CREATE TABLE IF NOT EXISTS processing_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entry_id UUID REFERENCES journal_entries(id) ON DELETE CASCADE,
    stage processing_stage NOT NULL,
    level VARCHAR(10) NOT NULL, -- debug, info, warn, error
    message TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_processing_logs_entry_id ON processing_logs(entry_id);
CREATE INDEX IF NOT EXISTS idx_processing_logs_stage ON processing_logs(stage);
CREATE INDEX IF NOT EXISTS idx_processing_logs_created_at ON processing_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_journal_entries_processing_stage ON journal_entries(processing_stage);

-- Create function to get stage duration
CREATE OR REPLACE FUNCTION get_stage_duration(entry_id UUID, stage processing_stage) 
RETURNS INTERVAL AS $$
DECLARE
    start_time TIMESTAMP WITH TIME ZONE;
    end_time TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Get the first log entry for this stage
    SELECT created_at INTO start_time
    FROM processing_logs
    WHERE processing_logs.entry_id = get_stage_duration.entry_id 
    AND processing_logs.stage = get_stage_duration.stage
    ORDER BY created_at ASC
    LIMIT 1;
    
    -- Get the first log entry for the next stage or completion
    SELECT created_at INTO end_time
    FROM processing_logs
    WHERE processing_logs.entry_id = get_stage_duration.entry_id 
    AND processing_logs.stage > get_stage_duration.stage
    ORDER BY created_at ASC
    LIMIT 1;
    
    -- If no next stage, check if processing is complete
    IF end_time IS NULL THEN
        SELECT processing_completed_at INTO end_time
        FROM journal_entries
        WHERE id = get_stage_duration.entry_id;
    END IF;
    
    -- Calculate duration
    IF start_time IS NOT NULL AND end_time IS NOT NULL THEN
        RETURN end_time - start_time;
    ELSE
        RETURN NULL;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create view for processing statistics
CREATE OR REPLACE VIEW processing_stats AS
SELECT 
    stage,
    COUNT(DISTINCT entry_id) as entries_count,
    AVG(EXTRACT(EPOCH FROM get_stage_duration(entry_id, stage))) as avg_duration_seconds,
    MAX(EXTRACT(EPOCH FROM get_stage_duration(entry_id, stage))) as max_duration_seconds,
    MIN(EXTRACT(EPOCH FROM get_stage_duration(entry_id, stage))) as min_duration_seconds
FROM processing_logs
GROUP BY stage;
`
