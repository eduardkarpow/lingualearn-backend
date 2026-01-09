CREATE TABLE subtitles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID,
    subtitle_key VARCHAR(255),
    shift INTEGER
);

CREATE INDEX idx_subtitles_video_id ON subtitles(video_id);