package main

var (
	// WMVHeader = 30 26 B2 75 8E 66 CF 11 A6 D9 00 AA 00 62 CE 6C
	WMVHeader = []byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}
)

// we use the same technique as in oggFix, and for files that dont have any part of the header we just put the header at the start of the file and hope for the best
// if this dont work, we try to find a way to reconstruct wmv files from a tool someone made
// if this dont work we can record the original game window
