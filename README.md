# Live Adaptive Video Speech Recognition

This is a proof-of-concept (PoC) that demonstrates two different end-to-end implementations of auto-generated subtitles sourced from chunks of MP4 video data.

## Strategies
- There are examples of two implementations here - a client-focused one and a server-focused one.

**The recommended strategy is a server-focused one** as it requires less processing and bandwidth usage for all parties.

A server-focused strategy is one that has direct access to the encoder or only its output for augmentation. Audio data is retrieved directly from the encoder output where it is then sent for transcription. Delivery of the transcripts can be performed in a number of ways, the most spec-compliant way would be live `WebVTT` segments.

A client-focused strategy can be implemented on any playback source but still has a small backend component in play. Audio data is sent from the client's browser to the backend component where it is then sent for transcription. Once the timed transcription is recieved, it is then translated to `WebVTT` cues to allow native rendering capabilities offered by the browser.

See *Known Issues* below for details on current limitations/issues
See *Roadmap* for future tasks/wishlist items

## Tech Used
- [hls.js](https://github.com/video-dev/hls.js) for browser HLS playback, and used to access remuxed chunks of mp4 data (for client-focused strategy)
- [Golang](https://golang.org/) for server side components
- [FFmpeg](https://ffmpeg.org/) for media transcoding
- [Google Cloud Speech-to-Text API](https://cloud.google.com/speech-to-text/) - receives the chunks of mp4 data for transcription

## Instructions
It's required to have a GCP service account setup
https://cloud.google.com/video-intelligence/docs/common/auth

Tested on macOS 10.14 Mojave w/ `ffmpeg` 4.1.1

- Place `ffmpeg` binary in a new directory `bin`
- Ensure environment varible `FFMPEG_PATH` is set in shell scripts
- Ensure Golang code dependencies are resolved:
```sh
$ go get -u cloud.google.com/go/speech/apiv1
$ go get -u github.com/fsnotify/fsnotify
$ go get -u github.com/gorilla/websocket  # for client-strategy only
$ go get -u google.golang.org/genproto/googleapis/cloud/speech/v1
```

- Retrieve a GCP service account and place under `gcp`
- Ensure path to GCP service account json file is correct in scripts

- For client strategy:
  - Invoke `./run.sh` in both `client` and `server` directories

- For server strategy:
  - Provide playback source URL in `server/encoder.go`
  - Invoke `./run.sh` in strategy root and `run.sh`  in `server` directory

## Known Issues
- (example) Archived Client Strategy - synchronization of translated `WebVTT` cues to timing in video may be offset temporarily

## Roadmap
- Live WebVTT support for server strategy example, currently using a custom delivery method for initial phase.
Pending resolution of: https://github.com/jwplayer/hls.js/pull/192
- Pass data with `ffmpeg` via stdin/out rather than writing/reading to disk
- Allow `ffmpeg` arguments to be provided via external source
- Two-phase transcription process that optionally translates the given transcript text to another language
- Integrate [Mozilla DeepSpeech](https://github.com/mozilla/DeepSpeech) provider, [pending work on exposing timed word offsets in audio](https://discourse.mozilla.org/t/speech-to-text-json-result-with-time-per-word/32681)

## Notes
- You may need to tweak the encoding settings for compatibility and/or optimal performance
