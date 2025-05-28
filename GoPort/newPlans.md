# New plan
I despise C++, i hate its syntax so much, and building it is a pain in the ass.
So going forward i will use go, since we only need to use 2d textures and play some videos, ebiten + ffmpeg will be enough for this project.

## I assume that
I didnt even test the current C++ code, am not sure if it really works, probably it is an abandoned project by a russian dev, either way this can be a fun experiment and not too hard to get started.

## Steps to make this plan work
Making a simple ebiten window that can play the videos that School Days have, getting the extractor working is the first step.
After getting the files, converting them to a codec that we can play inside ebiten.

- Make a Go School Days file extractor, the original code had some mentions of zlib so its compressed and encrypted
- Using ffmpeg, make a script to batch convert whatever the original files are to a format/container/codec we can use.
- We first make a simple game window that can play these videos.
- Then we add the subtitles.
- If all of the above works, we proceed to port the remaining functionaly of the original code.
