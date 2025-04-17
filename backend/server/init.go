package server

func (b *Board) InitBoard() {

	for y := 0; y < boardHeight; y++ {
		for x := 0; x < boardWidth; x++ {
			b.Pixels[y][x] = Pixel{
				R: 0,
				G: 0,
				B: 0,
			}
		}
	}

}
