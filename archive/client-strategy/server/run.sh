#!/bin/bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/video-speech-recognition/gcp/service-account.json"
export FFMPEG_PATH="/path/to/video-speech-recognition/bin/ffmpeg"

rm -rf tmp
go run main.go