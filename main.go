package main

import (
	"fmt"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Grain struct {
	pixelPos [2]float32
	boardPos [2]int32
	locked   bool
	moisture int
	yAcc     float32
	yVel     float32
	xAcc     float32
	xVel     float32
	mass     float32
}

func (g *Grain) draw(cellSize int) {
	baseR, baseG, baseB := 222, 161, 32

	r := uint8(baseR - int(g.moisture-1)*20)
	gColor := uint8(baseG - int(g.moisture-1)*15)
	b := uint8(baseB + int(g.moisture-1)*10)

	rl.DrawRectangle(int32(g.pixelPos[0]), int32(g.pixelPos[1]), int32(cellSize), int32(cellSize), rl.Color{R: r, G: gColor, B: b, A: 255})
}

func (g *Grain) addForce(force rl.Vector2) {
	yAcc := force.Y / g.mass
	aAcc := force.X / g.mass
	g.yAcc += yAcc
	g.xAcc += aAcc
}

func (g *Grain) update(cellSize int, sandPositions [][]int, bottomBuffer int) [][]int {
	if g.pixelPos[0] < 0 || g.pixelPos[0]+float32(cellSize) > float32(rl.GetScreenWidth()) {
		return sandPositions
	} // Check if the particle is within the screen

	if g.locked {
		return sandPositions
	} // Make sure the particle is not locked

	g.yAcc += 0.5 // Gravity not affected by mass
	g.yVel += g.yAcc
	g.xVel += g.xAcc

	stepSize := float32(cellSize) / 2 // Small step size to avoid skipping rows
	remainingFall := float32(g.yVel)  // Total movement this frame

	// Simulate movement in small steps
	for remainingFall > 0 {
		step := stepSize
		if remainingFall < stepSize {
			step = remainingFall
		}

		// Predict the next position
		nextPixelY := g.pixelPos[1] + step
		nextBoardY := int32(nextPixelY) / int32(cellSize)

		// Check for collision at the next grid position
		if nextBoardY >= int32(len(sandPositions)) {
			g.locked = true
			g.boardPos[1] = int32(len(sandPositions)) - 1
			g.pixelPos[1] = float32(g.boardPos[1]) * float32(cellSize)

			g.boardPos[0] = int32(g.pixelPos[0]) / int32(cellSize)
			g.pixelPos[0] = float32(g.boardPos[0]) * float32(cellSize)

			sandPositions[g.boardPos[1]][g.boardPos[0]] = 1
			return sandPositions
		}

		if sandPositions[nextBoardY][g.boardPos[0]] == 1 {
			g.locked = true
			g.boardPos[1] = nextBoardY - 1
			g.pixelPos[1] = float32(g.boardPos[1]) * float32(cellSize)

			g.boardPos[0] = int32(g.pixelPos[0]) / int32(cellSize)
			g.pixelPos[0] = float32(g.boardPos[0]) * float32(cellSize)

			sandPositions[g.boardPos[1]][g.boardPos[0]] = 1
			return sandPositions
		}

		// No collision, update position
		g.pixelPos[1] = nextPixelY
		g.boardPos[1] = nextBoardY

		g.pixelPos[0] += g.xVel
		g.boardPos[0] = int32(g.pixelPos[0]) / int32(cellSize)

		if g.boardPos[0] < 0 {
			g.boardPos[0] = 0
			g.pixelPos[0] = 0
			g.xVel = 0 // Stop movement at the boundary
		}
		if g.boardPos[0] >= int32(len(sandPositions[0])) {
			g.boardPos[0] = int32(len(sandPositions[0])) - 1
			g.pixelPos[0] = float32(g.boardPos[0]) * float32(cellSize)
			g.xVel = 0 // Stop movement at the boundary
		}
		remainingFall -= step
	}

	g.yAcc = 0
	return sandPositions
}

func main() {
	screenWidth, screenHeight := 800, 600

	// Initialize a small window for the resolution picker
	rl.InitWindow(400, 300, "Select Screen Dimensions")
	defer rl.CloseWindow()

	// Input fields
	widthInput := "800"
	heightInput := "600"
	currentInput := &widthInput // Pointer to track which input is being edited
	isRunning := true

	for isRunning && !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// Draw title
		rl.DrawText("Select Screen Dimensions", 50, 50, 20, rl.DarkGray)

		// Draw input labels
		rl.DrawText("Width:", 50, 100, 20, rl.Black)
		rl.DrawText("Height:", 50, 150, 20, rl.Black)

		// Draw input fields
		rl.DrawRectangle(120, 100, 200, 30, rl.LightGray)
		rl.DrawRectangle(120, 150, 200, 30, rl.LightGray)

		// Highlight active input
		if currentInput == &widthInput {
			rl.DrawRectangleLines(120, 100, 200, 30, rl.Blue)
		} else {
			rl.DrawRectangleLines(120, 150, 200, 30, rl.Blue)
		}

		rl.DrawText(widthInput, 125, 105, 20, rl.Black)
		rl.DrawText(heightInput, 125, 155, 20, rl.Black)

		// Instructions
		rl.DrawText("Press Tab to switch fields", 50, 220, 20, rl.Gray)

		// Handle keyboard input
		if rl.IsKeyPressed(rl.KeyTab) {
			if currentInput == &widthInput {
				currentInput = &heightInput
			} else {
				currentInput = &widthInput
			}
		}

		// Handle backspace
		if rl.IsKeyPressed(rl.KeyBackspace) && len(*currentInput) > 0 {
			*currentInput = (*currentInput)[:len(*currentInput)-1]
		}

		// Handle number input
		key := rl.GetKeyPressed()
		if key >= '0' && key <= '9' {
			*currentInput += string(rune(key))
		}

		// Start button
		if rl.IsKeyPressed(rl.KeyEnter) { // If Enter is pressed
			w, err1 := strconv.Atoi(widthInput)
			h, err2 := strconv.Atoi(heightInput)
			if err1 == nil && err2 == nil && w > 0 && h > 0 {
				screenWidth = w
				screenHeight = h
				isRunning = false // Exit the menu
			} else {
				rl.DrawText("Invalid input!", 50, 200, 20, rl.Red)
			}
		}

		rl.EndDrawing()
	}

	//
	//
	//
	//
	//
	//
	//
	//
	// Main game loop
	//
	//
	//
	//
	//
	//
	//
	//

	// Initialize the main game window with selected dimensions
	rl.InitWindow(int32(screenWidth), int32(screenHeight), "Complex Sand Simulation")
	defer rl.CloseWindow()

	var sandPositions [][]int
	var sandParticles []Grain
	buttonX := 0
	moistureSelected := 1 // Default dry

	var cellSize int = 10
	bottomBuffer := 100

	rl.SetTargetFPS(60)

	for i := 0; i < (rl.GetScreenHeight()-bottomBuffer)/cellSize; i++ {
		sandPositions = append(sandPositions, []int{})
		for j := 0; j < rl.GetScreenWidth()/cellSize; j++ {
			sandPositions[i] = append(sandPositions[i], 0)
		}
	}

	fmt.Println(len(sandPositions))
	fmt.Println(len(sandPositions[0]))

	for !rl.WindowShouldClose() { // Main Loop
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		// drawGrid(rl.GetScreenWidth(), rl.GetScreenHeight()-bottomBuffer, cellSize)

		for i := 0; i < len(sandParticles); i++ {
			sandParticles[i].draw(cellSize)
			// Wind
			var xForce float32 = float32(rl.GetRandomValue(-100, 100))
			xForce = xForce / 100000
			sandParticles[i].addForce(rl.NewVector2(xForce, 0))
			sandPositions = sandParticles[i].update(cellSize, sandPositions, bottomBuffer)
		} // Draw and update all particles

		bottomYPos := int32(rl.GetScreenHeight() - bottomBuffer)
		rightSide := int32(rl.GetScreenWidth())
		rl.DrawLine(0, bottomYPos, rightSide, bottomYPos, rl.White)

		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			// Check if the mouse is within the grid
			if rl.GetMousePosition().Y < float32(rl.GetScreenHeight()-bottomBuffer) {
				sandParticles = dropSand(sandParticles, cellSize, sandPositions, moistureSelected)
			}
		} // Drop sand

		// Slider
		value, xPos := slider(bottomBuffer, buttonX)
		buttonX = xPos
		moistureSelected = value

		rl.EndDrawing()
	}
}

func drawGrid(windowWidth, windowHeight, cellSize int) {
	var rows, cols int = windowHeight / cellSize, windowWidth / cellSize

	for i := 0; i < rows; i++ {
		y := i * cellSize
		startX := 0
		endX := windowWidth
		startPos := rl.NewVector2(float32(startX), float32(y))
		endPos := rl.NewVector2(float32(endX), float32(y))
		rl.DrawLineEx(startPos, endPos, 1, rl.Gray)
	}

	for i := 0; i < cols; i++ {
		x := i * cellSize
		startY := 0
		endY := windowHeight
		startPos := rl.NewVector2(float32(x), float32(startY))
		endPos := rl.NewVector2(float32(x), float32(endY))
		rl.DrawLineEx(startPos, endPos, 1, rl.Gray)
	}
}

func dropSand(sandParticles []Grain, cellSize int, sandPositions [][]int, moistureSelected int) []Grain {
	pos := rl.GetMousePosition()
	pos = rl.NewVector2(float32(int32(pos.X)/int32(cellSize)*int32(cellSize)), float32(int32(pos.Y)/int32(cellSize)*int32(cellSize)))

	// Check for overlapping with locked particles
	xIndex := int(pos.X / float32(cellSize))
	yIndex := int(pos.Y / float32(cellSize))

	if xIndex < 0 || xIndex >= len(sandPositions[0]) || yIndex < 0 || yIndex >= len(sandPositions) {
		return sandParticles
	}

	if sandPositions[yIndex][xIndex] == 1 {
		return sandParticles
	}
	sandParticles = append(sandParticles, Grain{
		pixelPos: [2]float32{pos.X, pos.Y},
		boardPos: [2]int32{int32(pos.X) / int32(cellSize), int32(pos.Y) / int32(cellSize)},
		locked:   false,
		yAcc:     1,
		yVel:     0,
		moisture: moistureSelected,
		mass:     0.1,
	})

	return sandParticles
}

func slider(topBuffer int, buttonX int) (int, int) {
	sliderRange := 10
	var pixelsPerValue int = 50
	var sliderLength int32 = int32(sliderRange * pixelsPerValue)
	var sliderWidth int32 = 20
	var sliderX int32 = int32(rl.GetScreenWidth()/2 - int(sliderLength/2))
	var sliderY int32 = int32(rl.GetScreenHeight()) - (int32(topBuffer) / 2)

	// Draw slider
	rl.DrawRectangle(sliderX, sliderY, sliderLength, sliderWidth, rl.Gray)

	// Slider button

	if buttonX == 0 {
		buttonX = int(sliderX)
	}

	buttonY := sliderY + sliderWidth/2

	radius := 17
	rl.DrawCircle(
		int32(buttonX),
		int32(buttonY),
		float32(radius),
		rl.White,
	)

	if rl.IsMouseButtonDown(rl.MouseLeftButton) {
		mousePos := rl.GetMousePosition()
		if mousePos.X >= float32(sliderX) && mousePos.X <= float32(sliderX+sliderLength) {
			if mousePos.Y > float32(sliderY)-float32(sliderWidth) {
				buttonX = int(mousePos.X)
			}
		}
	}

	// Calculate value
	value := (buttonX - int(sliderX)) / pixelsPerValue
	return value + 1, buttonX
}
