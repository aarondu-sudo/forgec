package internal

// capi:export
type Point struct {
	X int32
	Y int32
}

// capi:export
func Add(sum *Point, target *Point, addend *Point) (int32, *Point) {
	return 0, &Point{X: left.X + right.X, Y: left.Y + right.Y}
}

// capi:export
func Minus(left *Point, right *Point) (int32, *Point) {
	return 0, &Point{X: left.X - right.X, Y: left.Y - right.Y}
}
