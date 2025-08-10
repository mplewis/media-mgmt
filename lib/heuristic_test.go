package lib

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Video Stream Classification", func() {
	Describe("ClassifyVideoStreams", func() {
		Context("with a single video stream", func() {
			It("classifies it as primary", func() {
				streams := []Stream{
					{
						Index:     0,
						CodecType: "video",
						CodecName: "h264",
						Width:     1920,
						Height:    1080,
						Bitrate:   "5000000",
					},
				}

				result := ClassifyVideoStreams(streams, 3600.0)

				Expect(result.Primary).NotTo(BeNil())
				Expect(result.Auxiliary).To(HaveLen(0))
				Expect(result.Primary.CodecName).To(Equal("h264"))
			})
		})

		Context("with no video streams", func() {
			It("returns empty classification", func() {
				streams := []Stream{
					{
						Index:     0,
						CodecType: "audio",
						CodecName: "aac",
					},
				}

				result := ClassifyVideoStreams(streams, 3600.0)

				Expect(result.Primary).To(BeNil())
				Expect(result.Auxiliary).To(HaveLen(0))
			})
		})

		Context("with H.264 vs MJPEG streams", func() {
			It("prioritizes H.264 over MJPEG", func() {
				streams := []Stream{
					{
						Index:     0,
						CodecType: "video",
						CodecName: "mjpeg",
						Width:     160,
						Height:    120,
						Bitrate:   "50000",
					},
					{
						Index:     1,
						CodecType: "video",
						CodecName: "h264",
						Width:     1920,
						Height:    1080,
						Bitrate:   "5000000",
					},
				}

				result := ClassifyVideoStreams(streams, 3600.0)

				Expect(result.Primary).NotTo(BeNil())
				Expect(result.Primary.CodecName).To(Equal("h264"))
				Expect(result.Auxiliary).To(HaveLen(1))
				Expect(result.Auxiliary[0].CodecName).To(Equal("mjpeg"))
			})
		})

		Context("with same codec different resolutions", func() {
			It("prioritizes higher resolution", func() {
				streams := []Stream{
					{
						Index:     0,
						CodecType: "video",
						CodecName: "h264",
						Width:     640,
						Height:    480,
						Bitrate:   "1000000",
					},
					{
						Index:     1,
						CodecType: "video",
						CodecName: "h264",
						Width:     1920,
						Height:    1080,
						Bitrate:   "5000000",
					},
				}

				result := ClassifyVideoStreams(streams, 3600.0)

				Expect(result.Primary).NotTo(BeNil())
				Expect(result.Primary.Width).To(Equal(1920))
				Expect(result.Primary.Height).To(Equal(1080))
			})
		})

		Context("with complex real-world scenario", func() {
			It("prioritizes 4K HEVC over MJPEG cover art", func() {
				streams := []Stream{
					{
						Index:       0,
						CodecType:   "video",
						CodecName:   "mjpeg",
						Width:       600,
						Height:      900,
						Bitrate:     "200000",
						PixelFormat: "rgb24",
					},
					{
						Index:       2,
						CodecType:   "video",
						CodecName:   "hevc",
						Width:       3840,
						Height:      2160,
						Bitrate:     "15000000",
						PixelFormat: "yuv420p10le",
					},
				}

				result := ClassifyVideoStreams(streams, 7200.0)

				Expect(result.Primary).NotTo(BeNil())
				Expect(result.Primary.CodecName).To(Equal("hevc"))
				Expect(result.Primary.Width).To(Equal(3840))
				Expect(result.Auxiliary).To(HaveLen(1))
				Expect(result.Auxiliary[0].CodecName).To(Equal("mjpeg"))
			})
		})
	})


	Describe("getCodecScore", func() {
		It("scores codecs correctly", func() {
			Expect(getCodecScore("hevc")).To(Equal(100.0))
			Expect(getCodecScore("h264")).To(Equal(95.0))
			Expect(getCodecScore("av1")).To(Equal(90.0))
			Expect(getCodecScore("mjpeg")).To(Equal(10.0))
			Expect(getCodecScore("png")).To(Equal(5.0))
			Expect(getCodecScore("unknown")).To(Equal(50.0))
		})

		It("is case insensitive", func() {
			codecs := []string{"HEVC", "H264", "AV1", "MJPEG"}
			for _, codec := range codecs {
				upperScore := getCodecScore(codec)
				lowerScore := getCodecScore(strings.ToLower(codec))
				Expect(upperScore).To(Equal(lowerScore))
			}
		})
	})


	Describe("getIndexScore", func() {
		It("favors lower indices", func() {
			score0 := getIndexScore(0)
			score1 := getIndexScore(1)
			score5 := getIndexScore(5)

			Expect(score0).To(BeNumerically(">", score1))
			Expect(score1).To(BeNumerically(">", score5))
			Expect(score5).To(BeNumerically(">=", 0))
		})
	})


	Describe("getPixelFormatScore", func() {
		It("scores pixel formats correctly", func() {
			Expect(getPixelFormatScore("yuv420p")).To(Equal(15.0)) // yuv(10) + 420(5)
			Expect(getPixelFormatScore("yuv422p")).To(Equal(15.0)) // yuv(10) + 422(5)
			Expect(getPixelFormatScore("yuv444p")).To(Equal(15.0)) // yuv(10) + 444(5)
			Expect(getPixelFormatScore("rgb24")).To(Equal(-5.0))
			Expect(getPixelFormatScore("unknown")).To(Equal(0.0))
		})
	})


	Describe("parseBitrate", func() {
		It("parses bitrate from stream field", func() {
			stream := Stream{Bitrate: "5000000"}
			Expect(parseBitrate(stream)).To(Equal(int64(5000000)))
		})

		It("parses bitrate from BPS tag", func() {
			stream := Stream{
				Tags: map[string]string{"BPS": "3000000"},
			}
			Expect(parseBitrate(stream)).To(Equal(int64(3000000)))
		})

		It("prioritizes bitrate field over BPS tag", func() {
			stream := Stream{
				Bitrate: "5000000",
				Tags:    map[string]string{"BPS": "3000000"},
			}
			Expect(parseBitrate(stream)).To(Equal(int64(5000000)))
		})

		It("returns 0 when no bitrate info available", func() {
			stream := Stream{}
			Expect(parseBitrate(stream)).To(Equal(int64(0)))
		})

		It("returns 0 for invalid bitrate", func() {
			stream := Stream{Bitrate: "invalid"}
			Expect(parseBitrate(stream)).To(Equal(int64(0)))
		})
	})


	Describe("parseDurationTag", func() {
		It("parses valid duration formats", func() {
			Expect(parseDurationTag("01:30:45.500")).To(Equal(5445.5))
			Expect(parseDurationTag("00:05:30.000")).To(Equal(330.0))
			Expect(parseDurationTag("02:00:00.000")).To(Equal(7200.0))
		})

		It("returns 0 for invalid formats", func() {
			Expect(parseDurationTag("invalid")).To(Equal(0.0))
			Expect(parseDurationTag("10:30")).To(Equal(0.0))
		})
	})


	Describe("getDurationScore", func() {
		It("gives bonus for matching duration", func() {
			stream := Stream{
				Tags: map[string]string{"DURATION": "01:00:00.000"},
			}
			score := getDurationScore(stream, 3600.0)
			Expect(score).To(BeNumerically(">=", 15))
			Expect(score).To(BeNumerically("<=", 25))
		})

		It("penalizes very short duration", func() {
			stream := Stream{
				Tags: map[string]string{"DURATION": "00:00:01.000"},
			}
			score := getDurationScore(stream, 3600.0)
			Expect(score).To(BeNumerically(">=", -35))
			Expect(score).To(BeNumerically("<=", -25))
		})

		It("returns neutral for short format duration", func() {
			stream := Stream{
				Tags: map[string]string{"DURATION": "00:00:05.000"},
			}
			score := getDurationScore(stream, 5.0)
			Expect(score).To(BeNumerically(">=", -1))
			Expect(score).To(BeNumerically("<=", 1))
		})

		It("returns neutral when no duration tag", func() {
			stream := Stream{}
			score := getDurationScore(stream, 3600.0)
			Expect(score).To(BeNumerically(">=", -1))
			Expect(score).To(BeNumerically("<=", 1))
		})
	})


	Describe("calculateStreamScore integration", func() {
		It("scores main video higher than thumbnail", func() {
			mainStream := Stream{
				Index:       0,
				CodecName:   "h264",
				Width:       1920,
				Height:      1080,
				Bitrate:     "5000000",
				PixelFormat: "yuv420p",
			}

			thumbnailStream := Stream{
				Index:       1,
				CodecName:   "mjpeg",
				Width:       160,
				Height:      120,
				Bitrate:     "50000",
				PixelFormat: "rgb24",
			}

			mainScore := calculateStreamScore(mainStream, 3600.0)
			thumbScore := calculateStreamScore(thumbnailStream, 3600.0)

			Expect(mainScore).To(BeNumerically(">", thumbScore))
			Expect(mainScore).To(BeNumerically(">=", 0))
			Expect(mainScore).To(BeNumerically("<", 1000))
			Expect(thumbScore).To(BeNumerically("<", 100))
		})
	})


	Describe("extractVideoStreams", func() {
		It("filters only video streams", func() {
			streams := []Stream{
				{CodecType: "video", CodecName: "h264"},
				{CodecType: "audio", CodecName: "aac"},
				{CodecType: "video", CodecName: "mjpeg"},
				{CodecType: "subtitle", CodecName: "srt"},
			}

			videoStreams := extractVideoStreams(streams)

			Expect(videoStreams).To(HaveLen(2))
			for _, stream := range videoStreams {
				Expect(stream.CodecType).To(Equal("video"))
			}
		})
	})
})