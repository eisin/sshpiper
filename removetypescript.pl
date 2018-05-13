#!/usr/bin/perl
# https://unix.stackexchange.com/questions/14684/removing-control-chars-including-console-codes-colours-from-script-output
local $| = 1;
while (<>) {
    s/\r\n/\n/g;
    s/\r/\n/g;
    s/ \e[ #%()*+\-.\/]. |
       (?:\e\[|\x9b) [ -?]* [@-~] | # CSI ... Cmd
       (?:\e\]|\x9d) .*? (?:\e\\|[\a\x9c]) | # OSC ... (ST|BEL)
       (?:\e[P^_]|[\x90\x9e\x9f]) .*? (?:\e\\|\x9c) | # (DCS|PM|APC) ... ST
       #\e.|[\x80-\x9f] //xg;
       1 while s/[^\b][\b]//g;  # remove all non-backspace followed by backspace
    print;
}
