package strategies

import "os/exec"

// X264 ...
func X264(ffmpeg, source, segmentFilename, outputPath string) *exec.Cmd {

	cmd := exec.Command(
		ffmpeg,
		"-i", source,
		"-ignore_unknown",
		"-acodec", "aac",
		"-ar", "44100",
		"-ac", "2",
		"-async", "1",
		"-vsync", "-1",
		"-vcodec", "libx264",
		"-x264opts", "keyint=60:no-scenecut",
		"-profile:v", "high",
		"-level", "4.1",
		"-tune", "zerolatency",
		"-segment_list_flags", "live",
		"-flags", "+cgop",
		"-preset", "veryfast",
		"-bsf:a", "aac_adtstoasc",
		"-hls_segment_type", "fmp4",
		"-hls_time", "10",
		"-hls_list_size", "10",
		"-hls_flags", "delete_segments+omit_endlist",
		"-hls_segment_filename", segmentFilename,

		"-b:v:0", "1000k",
		"-s:v:0", "426x240",
		"-b:a:0", "64k",

		"-b:v:2", "2000k",
		"-s:v:2", "896x504",
		"-b:a:2", "192k",

		"-b:v:3", "5000k",
		"-s:v:3", "1280x720",
		"-b:a:3", "256k",

		"-map", "0:v", "-map", "0:a",
		"-map", "0:v", "-map", "0:a",
		"-map", "0:v", "-map", "0:a",

		"-var_stream_map", "v:0,a:0 v:1,a:1 v:2,a:2",
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
