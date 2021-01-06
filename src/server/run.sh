#!/bin/bash
export GOOGLE_APPLICATION_CREDENTIALS="/Users/mcunningham/Credentials/live-transcription-9e1584a26299.json"
export FFMPEG_PATH="/Users/mcunningham/Binaries/ffmpeg-osx_4.3.1"

# Removing old built binary if it exists
if [[ -f "./bin/vsr_x64" ]]; then
    echo "Previous built binary exists; removing..."
    rm ./bin/vsr_x64
fi

cd src/server
echo "Building VSR binary..."
go build -o ../../bin/vsr_x64
echo "Finished building."
../../bin/vsr_x64
echo "Server running".
