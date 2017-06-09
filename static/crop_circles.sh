#!/bin/bash

mkdir cropped;
mkdir out;

for FILE in orig/* ; do
    BASENAME="${FILE##*/}";
    BASENAME="${BASENAME%.*}";
    convert "$FILE" -resize "100^" -gravity center -crop 100x100+0+0 -strip "cropped/$BASENAME.png";
    convert "cropped/$BASENAME.png" -alpha on \( +clone -threshold -1 -negate -fill white -draw "circle 50,50 50,0" \) -compose copy_opacity -composite "out/$BASENAME.png";
done

