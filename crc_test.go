package atecc

import "testing"

func TestCrc16(t *testing.T) {
	// test cases from standard library: hash/crc32
	testCases := []struct {
		crc uint16
		in  string
	}{
		{0x0, ""},
		{0x8317, "a"},
		{0x159e, "ab"},
		{0x1ce9, "abc"},
		{0xe99c, "abcd"},
		{0x1da1, "abcde"},
		{0xa01a, "abcdef"},
		{0x9b97, "abcdefg"},
		{0x942e, "abcdefgh"},
		{0xae0f, "abcdefghi"},
		{0x8d13, "abcdefghij"},
		{0x574, "Discard medicine more than two years old."},
		{0xd6e4, "He who has a shady past knows that nice guys finish last."},
		{0xdd58, "I wouldn't marry him with a ten foot pole."},
		{0x9929, "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave"},
		{0xbfd5, "The days of the digital watch are numbered.  -Tom Stoppard"},
		{0xa772, "Nepal premier won't resign."},
		{0x386e, "For every action there is an equal and opposite government program."},
		{0xc41d, "His money is twice tainted: 'taint yours and 'taint mine."},
		{0xc14b, "There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977"},
		{0x526c, "It's a tiny change to the code and not completely disgusting. - Bob Manchek"},
		{0xfea6, "size:  a.out:  bad magic"},
		{0x3717, "The major problem is with sendmail.  -Mark Horton"},
		{0x9ed3, "Give me a rock, paper and scissors and I will move the world.  CCFestoon"},
		{0x4c0c, "If the enemy is within range, then so are you."},
		{0x2883, "It's well we cannot hear the screams/That we create in others' dreams."},
		{0xf868, "You remind me of a TV show, but that's all right: I watch it anyway."},
		{0x6348, "C is as portable as Stonehedge!!"},
		{0xcc63, "Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley"},
		{0x16ef, "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule"},
		{0xf73d, "How can you write a big system without C++?  -Paul Glick"},
	}

	for _, tc := range testCases {
		t.Run(tc.in, func(t *testing.T) {
			if crc := crc16([]byte(tc.in)); crc != tc.crc {
				t.Errorf("got %#x want %#x", crc, tc.crc)
			}
		})
	}
}
