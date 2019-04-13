class VideoSpeechRecognition {

  constructor (media, options) {
    let defaults = {
      mode               : VideoSpeechRecognition.Mode.Live,
      endpoint           : '',
      headers            : { },

      tracks: {
        metadata: {
          initial: 'hidden'
        },
        subtitles: {
          name: 'English (auto-generated)',
          lang: 'en',
          initial: 'showing'
        }
      }
    }

    this._onTick            = this._onTick.bind(this)
    this._onSocketOpened    = this._onSocketOpened.bind(this)
    this._onSocketClosed    = this._onSocketClosed.bind(this)
    this._onSocketMessage   = this._onSocketMessage.bind(this)
    this._onSocketError     = this._onSocketError.bind(this)

    this._options           = Object.assign({}, defaults, options)
    this._started           = false
    this._media             = media
    this._header            = null
    this._buffer            = []
    this._lastBufferLength  = 0
    this._isBusy            = false
    this._processInterval   = null
    this._textTracks        = {}
    this._socket            = null
    this._reqBufRange       = null
    this._processedBufRange = null
    this._processableChunks = []
  }

  static get Mode() {
    return {
      Live: 0,
      VOD : 1   // Will not prune buffer if used. Potential memory problems tho?
    }
  }

  processHeader (header) {
    // The initial segment of the fragmented audio mp4
    this._header = header
  }

  processData (data) {
    // Staggering invocations by 500ms to let the buffered range update...
    setTimeout(() => {
      let videoEl = this._media
      let range = {
        start: videoEl.buffered.start(0),
        end  : videoEl.buffered.end(0)
      }

      // Translating the range based on the buffer
      let lastRange = this._processedBufRange
      let shouldOffset = ( lastRange && ( range.end !== lastRange.end || range.start !== lastRange.start ) )

      if (shouldOffset)
        range.start = lastRange.end

      // Adding the data as a new entry, or appending to an existing typed array within the same range
      let sameRange = this._buffer.filter(b => b.range.start === range.start && b.range.end === range.end)
      sameRange = sameRange[0]

      if (!sameRange)
        this._buffer.push({ data, range })
      else
        sameRange.data = this._mergeTypedArrays(sameRange.data, data)

      this._processedBufRange = Object.assign({}, range)

    }, 500)
  }

  start () {
    this._createSocket()
    this._configureTextTracks()

    this._processInterval = setInterval(this._onTick, 1000)
    this._started = true
  }

  stop () {
    if (this._started) return
    clearInterval(this._processInterval)

    this._killSocket()
    this._cleanTracks()
    this._clearProcessableBuffer()

    this._reqBufRange = null
    this._processedBufRange = null
    this._processInterval = null
    this._started = false
  }

  _onTick () {
    if (this._isBusy) {
      let buffer = this._gatherProcessableBuffer()
      if (!buffer.data.length) {
        console.info('[VSR] onTick | busy, no processable buffer, wait')
        return  // nothing to do
      }

      // Adding the blob to the queue if busy
      console.info(`[VSR] onTick | busy, storing blob in processable queue, queued count:${this._processableChunks.length}`)
      this._processableChunks.push({
        queued: true,
        data: buffer.data,
        range: buffer.range
      })

      // The contents are safe in our secondary structured queue, the main buffer can be cleared to reclaim memory
      this._clearProcessableBuffer()
      return
    }

    let chunk = this._getPriorityProcessableItem()  // will be a queued item, or fresh data
    if (!chunk || !chunk.data.length) return  // nothing to do

    this._isBusy = true
    this._clearProcessableBuffer()

    if (chunk.queued)
      console.info(`[VSR] onTick | processing queued audio blob, queued count:${this._processableChunks.length}`)
    else
      console.info('[VSR] onTick | processing fresh audio blob')

    let contentType = this._options.headers['Content-Type']
    let blob = this._buildMP4BlobFromBuffer(chunk.data, contentType)

    console.info('[VSR] onTick | sending socket message containing audio blob')
    this._reqBufRange = Object.assign({}, chunk.range)
    this._socket.send(blob)
    // this._downloadBlobAsFile(blob, 'filename.mp4')
  }

  _getPriorityProcessableItem () {
    // Is there a queue?
    if (this._processableChunks.length > 0) {
      // Return the first queued chunk of data
      return this._processableChunks.shift()
    }

    // No queue, just fresh data, get the latest
    return this._gatherProcessableBuffer()
  }

  _createSocket () {
    let conn = new WebSocket(this._options.endpoint)

    conn.onclose = this._onSocketClosed
    conn.onerror = this._onSocketError
    conn.onmessage = this._onSocketMessage
    conn.onopen = this._onSocketOpened

    console.info('[VSR] createSocket | Socket created')

    this._socket = conn
  }

  _onSocketOpened (event) {
    console.info('[VSR] onSocketOpened | Socket opened, no longer busy')
    this._isBusy = false
  }

  _onSocketClosed (event) {
    console.info('[VSR] onSocketClosed | Re-opening socket, going to busy mode..')
    this._isBusy = true
    this._killSocket()
    this._createSocket()
  }

  _onSocketMessage (event) {
    let parsed = JSON.parse(event.data)

    if (parsed.hasOwnProperty('fail')) {
      // Failed to parse server-side, so we'll ignore it for now.
      // TODO add logic to retry that failed chunk of audio
      this._isBusy = false
      console.info('[VSR] onSocketMessage | failed transcription, ignoring...')
      return
    }

    // This parsed data belongs to buffered range { start: _reqBufRange.start, end: _reqBufRange.end }
    this._translateTranscriptToCues(parsed, this._reqBufRange)
    console.info(`
      [VSR] onSocketMessage | translated and added cues from transcript for buffered range
      start: ${this._reqBufRange.start} | end: ${this._reqBufRange.end}
    `)

    if (this._options.mode === VideoSpeechRecognition.Mode.Live) {
      console.info('[VSR] onSocketMessage | live mode, pruning old cues')
      this._pruneOOWCues()
    }

    this._isBusy = false
  }

  _onSocketError (event) {
    console.error('[VSR] onSocketError | unexpected error', event)
  }

  _killSocket () {
    if (this._socket) {
      this._socket.close()
      this._socket = null
    }
  }

  _translateTranscriptToCues (recognitionResponse, range) {
    if (recognitionResponse && !recognitionResponse.hasOwnProperty('results')) return

    // Flattening down to a single dimensional array of transcript objects
    let transcripts = recognitionResponse.results
      .map(report => report.alternatives)
      .reduce((acc, val) => acc.concat(val), [])

    const VTTCue = window.VTTCue || window.TextTrackCue

    // TODO: more intelligent grouping, should be done by the
    //       tiny delta between start and end times of nanosecond precision

    let numPerGroup = 10
    let cueGroups = []

    transcripts.forEach(transcript => {
      this._processConfidenceScore(transcript.confidence, range)

      let cues = transcript.words
        .map(structuredWord => {
          let start = range.start + this._fromStructuredNanoTime(structuredWord.start_time)
          let end = range.start + this._fromStructuredNanoTime(structuredWord.end_time)

          return { start, end, word: structuredWord.word }
        })

      let i = 0
      let cuesCount = cues.length
      // let expectedNumOfGroups = Math.floor( cuesCount / numPerGroup )
      let group = []

      do {
        let cue = cues.shift()
        let next = cues[0]

        if (cue)
          group.push(cue)

        let isLast = ( cue && !next )

        if (!cue || isLast || group.length === numPerGroup) {
          let groupClone = group.slice(0)
          cueGroups.push(groupClone)
          group.length = 0
        }

        if (!cue)
          break

        i++

      } while (i < cuesCount)
    })

    // Averaging out the start/end times to group words together, as nanosecond accuracy causes words to appear 1-by-1 very quickly
    cueGroups.forEach(group => {
      let firstCueStartTime = group[0].start
      let lastCueEndTime = group[ group.length - 1 ].end

      let contents = group
        .map(cue => cue.word)
        .join(' ')

      let combinedCue = new VTTCue(firstCueStartTime, lastCueEndTime, contents)
      this._textTracks.subtitles.addCue(combinedCue)
    })
  }

  _processConfidenceScore (score, range) {
    // Creating a metadata cue for this range
    const VTTCue = window.VTTCue || window.TextTrackCue

    let cue = new VTTCue(range.start, range.end, JSON.stringify({
      confidence: score,
      range
    }))

    this._textTracks.metadata.addCue(cue)
  }

  _fromStructuredNanoTime (struct) {
    let seconds = struct.seconds
    let nanos = struct.nanos

    let result = 0

    if (seconds)
      result = seconds

    if (nanos)
      // e.g. Translating 4000000 nanos -> 0.4 seconds
      result += ( ( nanos / 1000000 ) / 1000 )

    return result
  }

  /**
   * Returns the buffered content in a structured manner as a shallow clone
   */
  _gatherProcessableBuffer () {
    let processable = this._buffer.slice(0)

    let range = { start: 0, end: 0 }
    let data = processable.map(b => b.data)

    let firstEntry = processable[0]
    let lastEntry = processable[ data.length - 1 ]

    if (firstEntry)
      range.start = firstEntry.range.start

    if (lastEntry)
      range.end = lastEntry.range.end
    else
      range.end = range.start

    return { range, data, queued: false }
  }

  _clearProcessableBuffer () {
    this._buffer.length = 0
  }

  /**
   * Removes cues that are outside the sliding window ( plus a configurable offset )
   */
  _pruneOOWCues () {
    // TODO
  }

  _buildMP4BlobFromBuffer (buf, type) {
    let uint8 = this._arrayConcat(
      [ this._header ].concat(buf)
    )

    // Returning a playable file that could be downloaded
    return new Blob([ uint8 ], { type })
  }

  _downloadBlobAsFile (blob, filename) {
    let anchor, url

    anchor = document.createElement('a')
    anchor.style = 'display: none'

    document.body.appendChild(anchor)

    url = window.URL.createObjectURL(blob)
    anchor.href = url
    anchor.download = filename
    anchor.click()

    window.URL.revokeObjectURL(url)
    document.body.removeChild(anchor)
  }

  _configureTextTracks () {
    const trackOptions = this._options.tracks
    const tracks = this._textTracks
    const isPreconfigured = Object.keys(tracks).length > 0

    if (!isPreconfigured) {
      tracks.metadata = videoEl.addTextTrack('metadata')
      tracks.subtitles = videoEl.addTextTrack('subtitles', trackOptions.subtitles.name, trackOptions.subtitles.lang)
    }

    tracks.metadata.mode = trackOptions.metadata.initial
    tracks.subtitles.mode = trackOptions.subtitles.initial
  }

  _cleanTracks () {
    Object
      .values(this._textTracks)
      .forEach(track => {
        do {
          track.removeCue(track.cues[0])
        } while (typeof track.cues[0] !== 'undefined')

        track.mode = 'hidden'
      })
  }

  _mergeTypedArrays (base, toAppend) {
    let merged = new Uint8Array(base.length + toAppend.length)
    merged.set(base)
    merged.set(toAppend, base.length)

    return merged
  }

  _arrayConcat (inputArray) {
    let totalLength = inputArray.reduce((prev, curr) => prev + curr.length, 0)
    let result = new Uint8Array(totalLength)
    let offset = 0

    inputArray.forEach(element => {
      result.set(element, offset)
      offset += element.length
    })

    return result
  }

}
