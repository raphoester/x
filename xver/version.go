package xver

type Version struct {
	seq int
	mod bool
}

func Restore(stored int) *Version {
	return &Version{
		seq: stored,
		mod: false,
	}
}

func New() *Version {
	return &Version{
		seq: 0,
		mod: true,
	}
}

func (v *Version) RecordNewModification() {
	if v.mod {
		return
	}

	v.seq++
	v.mod = true
}

func (v *Version) Current() int {
	return v.seq
}

func (v *Version) Modified() bool {
	return v.mod
}

func (v *Version) Previous() int {
	if v.seq == 0 {
		return 0
	}

	return v.seq - 1
}

// Loaded returns the version as it was when the object was loaded in memory.
//
// If Current() == 0, the object has just been created and was never stored.
// Loaded will return -1 in this case.
func (v *Version) Loaded() int {
	if v.mod {

		return v.seq - 1
	}

	return v.seq
}
