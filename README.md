# Live Adaptive Video Speech Recognition

This is a proof-of-concept (PoC) that demonstrates two different end-to-end implementations of auto-generated subtitles sourced from an HLS Live Stream.

This PoC was referenced in the blog post "_Live Adaptive Video Speech Recognition_" on [REDspace](https://redspace.com)'s blog [Well Red](https://medium.com/well-red/live-adaptive-video-speech-recognition-21345ff380e9).


## Strategies
There are examples of two implementations here - a client-focused one (under `./archive/`) and a server-focused one (under `./src`).

A server-focused strategy is one that has direct access to the encoder or only its output for augmentation. Audio data is retrieved directly from the encoder output where it is then sent for transcription. Delivery of the transcripts can be performed in a number of ways, the most spec-compliant way would be live `WebVTT` segments.

A client-focused strategy can be implemented on any playback source but still has a small backend component in play. Audio data is sent from the client's browser to the backend component where it is then sent for transcription. Once the timed transcription is recieved, it is then translated to `WebVTT` cues to allow native rendering capabilities offered by the browser. An early PoC of this strategy can be found under `./archive`.

## Content Differences
This PoC is designed to demonstrate live content, but it can be applied to work with VoD as well.

------

See *Known Issues* below for details on current limitations/issues

See *Roadmap* for future tasks/wishlist items

## Tech Used
- [hls.js](https://github.com/video-dev/hls.js)
- [Golang](https://golang.org/)
- [FFmpeg](https://ffmpeg.org/)
- [Google Cloud Speech-to-Text API](https://cloud.google.com/speech-to-text/)

## Instructions
It's required to have a GCP service account setup
https://cloud.google.com/video-intelligence/docs/common/auth

Tested on:
- macOS 10.14 Mojave w/ `ffmpeg` 4.1.1
- Windows 10 x64 w/ `ffmpeg` 4.1.1

Steps:
1) Place `ffmpeg` binary in a new directory `bin`
2) Ensure environment varibles `GOOGLE_APPLICATION_CREDENTIALS` and `FFMPEG_PATH` are set in shell scripts
3) Retrieve a GCP service account JSON file and place under `gcp`

  - Provide playback source URL in `src/server/encoder/controller.go`
  - (optional) tweak encoding profile (e.g. x264 -> `src/server/encoder/strategies/x264.go`)
  - Run `run-client.sh` and `run-server.sh`

## Known Issues
- Error handling - more testing needed, could crash the application

## Roadmap
- Integration of Microsoft's Speech-to-Text API, see issue https://github.com/michaelcunningham19/video-speech-recognition/issues/2
- Live WebVTT support for server strategy example, currently using a custom delivery method for initial phase.
- Pass data with `ffmpeg` via stdin/out rather than writing/reading to disk
- Allow `ffmpeg` arguments to be provided via external source
- First class VOD support
- Published go module
- Two-phase transcription process that optionally translates the given transcript text to another language
- Integrate [Mozilla DeepSpeech](https://github.com/mozilla/DeepSpeech) provider, [pending work on exposing timed word offsets in audio](https://discourse.mozilla.org/t/speech-to-text-json-result-with-time-per-word/32681)

## Notes
- You may need to tweak the encoding settings for compatibility and/or optimal performance
