# Live Adaptive Video Speech Recognition

This is a proof-of-concept (PoC) that demonstrates two different end-to-end implementations of auto-generated subtitles sourced from an HLS Live Stream.

This PoC was referenced in the blog post "_Live Adaptive Video Speech Recognition_" on [REDspace](https://redspace.com)'s blog [Well Red](https://medium.com/well-red/live-adaptive-video-speech-recognition-21345ff380e9).


## Strategies
A server-focused strategy is the preferred strategy that assumes you have direct access to the encoder (or only its output) for augmentation. Audio data is retrieved directly from the encoder output where it is then sent for transcription. Delivery of the transcripts can be performed in a number of ways, the most spec-compliant way would be live `WebVTT` segments.

A secondary strategy is a client-focused one. Which can be implemented on any playback source but still has a small backend component in use. Audio data is sent from the client's browser to the backend component where it is then sent for transcription. Once the timed transcription is recieved, it is then translated to `WebVTT` cues to allow native rendering capabilities offered by the browser. A very early PoC of this strategy can be found under `./archive` and is not recommended for use.

## Content Differences
This PoC is designed to demonstrate live content, but it can be applied to work with VoD as well (either on-the-fly or once)

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
- macOS 11.1 Big Sur w/ `ffmpeg` 4.3.1
- Windows 10 x64 w/ `ffmpeg` 4.3.1

Steps:
1) Place `ffmpeg` binary in a new directory `bin`
2) Ensure environment varibles `GOOGLE_APPLICATION_CREDENTIALS` and `FFMPEG_PATH` are set in shell scripts
3) Retrieve a GCP service account JSON file and place under `gcp`

  - Provide playback source URL in `src/server/encoder/controller.go`
  - (optional) tweak encoding profile (e.g. x264 -> `src/server/encoder/strategies/x264.go`)
  - Run `npm run start:client` and `npm run start:server`

## Known Issues
- Error handling - more testing needed, could crash the application

## Roadmap
- Integration of Microsoft's Speech-to-Text API, see issue https://github.com/michaelcunningham19/video-speech-recognition/issues/2
- Live WebVTT support for server strategy example, currently using a custom delivery method for initial phase.
- Allow `ffmpeg` arguments to be provided via external source
- First class VOD support
- Published go module
- Two-phase transcription process that optionally translates the given transcript text to another language
- Integrate [Mozilla DeepSpeech](https://github.com/mozilla/DeepSpeech) provider, [pending work on exposing timed word offsets in audio](https://discourse.mozilla.org/t/speech-to-text-json-result-with-time-per-word/32681)

## Notes
- You may need to tweak the encoding settings for compatibility and/or optimal performance
