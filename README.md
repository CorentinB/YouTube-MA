[![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)](https://forthebadge.com)[![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)](https://forthebadge.com) [![Build Status](https://travis-ci.org/CorentinB/YouTube-MA.svg?branch=master)](https://travis-ci.org/CorentinB/YouTube-MA) [![Go Report Card](https://goreportcard.com/badge/github.com/CorentinB/youtube-ma)](https://goreportcard.com/report/github.com/CorentinB/youtube-ma) [![Codacy Badge](https://api.codacy.com/project/badge/Grade/e4ff7d9036f24567a03ff592868c366b)](https://www.codacy.com/project/CorentinB/youtube-ma/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=CorentinB/youtube-ma&amp;utm_campaign=Badge_Grade_Dashboard)
# YouTube-MA
💾 Light and fast YouTube metadata archiver written in Golang

# Usage

First download the latest release from https://github.com/CorentinB/youtube-ma/releases
Make it executable with:
```
chmod +x youtube-ma
```

Then here is an example of usage with a single ID:
```
./youtube-ma MPBfVp0tB8E
```
But you can also use a list of IDs, be carefull to have an ID per line, no complete URL.
```
./youtube-ma my_list.txt 32
```
Here **32** is the number of goroutines maximum that can be run at the same time, it'll depend on your system, as it's also linked to a certain number of files opened at the same time, that could be limited by your system's configuration. If you want to use a bigger value, tweak your system, such as **ulimit**.
Default for this value if you don't precise any value is **16**, should be safe in most system.

# Example

![example](https://image.noelshack.com/fichiers/2018/30/3/1532529549-selection-355.png)