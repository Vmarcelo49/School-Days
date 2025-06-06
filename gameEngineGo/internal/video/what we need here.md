# Requirements

We need to convert WMV to MPEG because ebiten currently only plays mpeg videos.
So we will use ffmpeg in PATH to convert the videos.

we need to implement our video logic just like the cpp code does.

we will be limited so we only will have these functions for the video player

load(videoData []byte) -> loads a video into the filesys
play(videoName string)
pause()
stopVideo()

reference for the ebiten video
[https://github.com/hajimehoshi/ebiten/tree/main/examples/video](https://github.com/hajimehoshi/ebiten/tree/main/examples/video)
