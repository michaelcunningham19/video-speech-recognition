package strategies

import "os/exec"

// NVENC ...
func NVENC(ffmpeg, source, segmentFilename, outputPath string) *exec.Cmd {

	cmd := exec.Command(
		ffmpeg,
		// "-loglevel", "debug",
		"-ignore_unknown",

		"-i", source,

		"-vcodec", "h264_nvenc",
		"-preset", "slow",
		"-profile:v", "high",
		"-level", "4.1",

		"-flags", "+cgop",

		"-acodec", "aac",
		"-ar", "44100",
		"-ac", "2",
		"-async", "1",
		"-bsf:a", "aac_adtstoasc",

		"-segment_list_flags", "live",

		"-hls_segment_type", "fmp4",
		"-hls_time", "10",
		"-hls_list_size", "10",
		"-hls_flags", "delete_segments+omit_endlist",
		"-hls_segment_filename", segmentFilename,

		"-b:v:0", "1500k",
		"-s:v:0", "896x504",
		"-b:a:0", "192k",

		"-b:v:1", "3000k",
		"-s:v:1", "1280x720",
		"-b:a:1", "256k",

		"-map", "0:v", "-map", "0:a",
		"-map", "0:v", "-map", "0:a",

		"-var_stream_map", "v:0,a:0 v:1,a:1",
		"-master_pl_name", "master.m3u8",
		"-master_pl_publish_rate", "1",
		"-hide_banner",
		"-reconnect_at_eof",
		"-reconnect_streamed",
		"-reconnect_delay_max", "3",

		outputPath,
	)

	return cmd

}
