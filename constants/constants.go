package constants

import "regexp"

// .torrent file magic number.
// See: https://en.wikipedia.org/wiki/Torrent_file , https://en.wikipedia.org/wiki/Bencode .
const TORRENT_FILE_MAGIC_NUMBER = "d8:announce"

// 个别种子没有 announce / announce-list 字段，第一个字段是 creation date。这类种子可以通过 DHT 下载成功。
const TORRENT_FILE_MAGIC_NUMBER2 = "d13:creation date"

const FILENAME_INVALID_CHARS_REGEX = `[<>:"/\|\?\*]+`
const PERM = 0600 // 程序创建的所有文件的 PERM

var FilenameInvalidCharsRegex = regexp.MustCompile(FILENAME_INVALID_CHARS_REGEX)

const FILENAME_SUFFIX_ADDED = ".added"
const FILENAME_SUFFIX_FAILED = ".failed"
const FILENAME_SUFFIX_DOWNLOADED = ".downloaded"

// Some ptool cmds add a suffix to processed torrent filenames.
// These cmds are hardcoded to ignore files with these suffixes (to prevent process of already processed torrent),
// even if it's specified by "*" wildcard pattern arg.
// Current Values: [".added", ".failed", ".downloaded"].
var ProcessedTorrentFilenameSuffixes = []string{
	FILENAME_SUFFIX_ADDED,
	FILENAME_SUFFIX_FAILED,
	FILENAME_SUFFIX_DOWNLOADED,
}
