# Transmission Torrent Cleaner [![Build Status](https://travis-ci.org/kalbasit/transmission-torrent-cleaner.svg?branch=master)](https://travis-ci.org/kalbasit/transmission-torrent-cleaner)

It's a simple Go program that monitors your transmission and can
optionally remove finished or stalled torrents.

You can optionally skip a torrent from the removal by passing the
a text/template. The template will receive the
[torrent](https://github.com/odwrtw/transmission/blob/3b39d734964d4b2b61267979feb8b5d0a2dc9a23/torrent.go#L123-L193)
as data and must result in `true` if the torrent must be skipped.
Everything else is considered as `do not skip`.

## Example

The following monitors a tranmission running on 192.168.1.105 port 9091
and removes stalled and finished torrents unless the
[DownloadDir](https://github.com/odwrtw/transmission/blob/3b39d734964d4b2b61267979feb8b5d0a2dc9a23/torrent.go#L134)
is set to `/nas/Downloads`.

```shell
$ docker run -d kalbasit/transmission-torrent-cleaner \
  -transmission-url="http://192.168.1.105:9091/transmission/rpc" \
  -remove-finished \
  -remove-stalled \
  -ignore-template '{{ if eq .DownloadDir "/nas/Downloads" }}true{{ end }}'
```

## Credits

This work is inspired by Albert's [excellent transmission docker
image](https://github.com/albertrdixon/docker-transmission/blob/ad71c8dd417572c73455e42876469442f0bf9c76/scripts/torrent_cleaner.py)

## Licenses

All source code is licensed under the [MIT License](https://raw.github.com/kalbasit/transmission-torrent-cleaner/master/LICENSE).
