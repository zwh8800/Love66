#!/bin/sh

go run main.go 156277 - 2>log | mplayer -rawaudio samplesize=2:channels=2:rate=48000 -demuxer rawaudio -
