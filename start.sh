go run main.go 431179 - 2>log | mplayer -rawaudio samplesize=2:channels=2:rate=48000 -demuxer rawaudio -
