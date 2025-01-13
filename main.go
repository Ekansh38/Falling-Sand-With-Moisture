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
	xPush    float32
	yPush    float32
	xVPush   float32
	yVPush   float32
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

func (g *Grain) windPush(force rl.Vector2) {
	yAcc := force.Y / g.mass
	aAcc := force.X / g.mass
	g.yPush += yAcc
	g.xPush += aAcc
}

func checkColumnStable(x int32, sandPositions [][]int, sandParticles []Grain, cellSize int) bool {
	for y := len(sandPositions) - 1; y >= 0; y-- { // Iterate from bottom to top
		if sandPositions[y][x] == 1 { // Particle exists at this position
			// Find the particle in the list
			for i := 0; i < len(sandParticles); i++ {
				if sandParticles[i].boardPos[0] == x && sandParticles[i].boardPos[1] == int32(y) {
					// Check if it's stable
					if !sandParticles[i].locked && y < len(sandPositions)-1 {
						if sandPositions[y+1][x] == 0 { // No support below
							return false
						}
					}
				}
			}
		}
	}
	return true
}

func (g *Grain) unlock(sandPositions [][]int, sandParticles []Grain) {
	g.locked = false
	g.yVel = 1 // Reset velocity to start falling
	for i := 0; i < len(sandParticles); i++ {
		if sandParticles[i].boardPos[0] == g.boardPos[0] && sandParticles[i].boardPos[1]+1 == g.boardPos[1] {
			sandParticles[i].unlock(sandPositions, sandParticles)
		}
	}
}

func (g *Grain) update(cellSize int, sandPositions [][]int, bottomBuffer int, sandParticles []Grain) [][]int {
	if g.locked {
		g.moisture = 1
	} else {
		g.moisture = 10
	}

	g.boardPos[0] = int32(g.pixelPos[0]) / int32(cellSize)
	g.boardPos[1] = int32(g.pixelPos[1]) / int32(cellSize)

	// Apply wind forces
	g.yVPush += g.yPush
	g.xVPush += g.xPush

	newPixelPos := rl.NewVector2(g.pixelPos[0]+g.xVPush, g.pixelPos[1]+g.yVPush)
	newBoardPos := [2]int32{int32(newPixelPos.X) / int32(cellSize), int32(newPixelPos.Y) / int32(cellSize)}

	if g.boardPos[0] != newBoardPos[0] || g.boardPos[1] != newBoardPos[1] {
		g.unlock(sandPositions, sandParticles)
		sandPositions[g.boardPos[1]][g.boardPos[0]] = 0
	}

	g.pixelPos[1] += g.yVPush
	g.pixelPos[0] += g.xVPush

	g.boardPos[1] = int32(g.pixelPos[1]) / int32(cellSize)
	g.boardPos[0] = int32(g.pixelPos[0]) / int32(cellSize)

	// Reduce push gradually
	g.yVPush *= 0.9
	g.xVPush *= 0.9
	g.yPush = 0
	g.xPush = 0

	// Ensure particle is within bounds
	if g.pixelPos[0] < 0 || g.pixelPos[0]+float32(cellSize) > float32(rl.GetScreenWidth()) {
		return sandPositions
	}

	// Skip locked particles
	if g.locked {
		return sandPositions
	}

	// Gravity
	g.yAcc = 10 // Gravity not affected by mass
	g.yVel += g.yAcc
	g.xVel += g.xAcc

	remainingFall := float32(g.yVel)

	// Simulate falling
	for remainingFall > 0 {
		step := float32(cellSize) / 2
		if remainingFall < step {
			step = remainingFall
		}

		nextPixelY := g.pixelPos[1] + step
		nextBoardY := int32(nextPixelY) / int32(cellSize)

		// Check for collision below
		if nextBoardY >= int32(len(sandPositions)) || sandPositions[nextBoardY][g.boardPos[0]] == 1 {
			// Check column stability
			if checkColumnStable(g.boardPos[0], sandPositions, sandParticles, cellSize) {
				g.locked = true
				g.boardPos[1] = nextBoardY - 1
				g.pixelPos[1] = float32(g.boardPos[1]) * float32(cellSize)

				g.boardPos[0] = int32(g.pixelPos[0]) / int32(cellSize)
				g.pixelPos[0] = float32(g.boardPos[0]) * float32(cellSize)

				sandPositions[g.boardPos[1]][g.boardPos[0]] = 1
				return sandPositions
			} else {
				// Particle remains unlocked
				break
			}
		}

		g.pixelPos[1] = nextPixelY
		g.boardPos[1] = nextBoardY
		remainingFall -= step
	}

	g.yAcc = 0
	return sandPositions
}

type Fan struct {
	pos       rl.Vector2
	direction int
	strength  float32
	width     float32
	height    float32
	wind      []Wind
	timer     int
	spawnRate int
}

func (f *Fan) draw(cellSize int) {
	rl.DrawRectangle(int32(f.pos.X), int32(f.pos.Y), int32(cellSize), (int32(cellSize) * 8), rl.Color{R: 0, G: 0, B: 255, A: 255})
}

func (f *Fan) update(cellSize int, sandPositions [][]int, sandParticles []Grain) {
	f.timer++ // Increment the timer

	if f.timer >= f.spawnRate {
		// Emit wind particles
		xPos := f.pos.X + f.width
		for i := 0; i < 1; i++ { // Emit two wind particles
			yPos := rl.GetRandomValue(int32(f.pos.Y), int32(f.pos.Y+f.height))
			direction := 4
			if f.pos.X > float32(rl.GetScreenWidth()/2) {
				direction = 2
			}
			wind := Wind{
				direction: direction,
				xVel:      2.0, // Constant velocity
				yVel:      0.0,
				xPos:      float32(xPos),
				yPos:      float32(yPos), // Small offset for multiple particles
			}
			f.wind = append(f.wind, wind)
		}
		f.timer = 0 // Reset the timer
	}
	for i := 0; i < len(f.wind); i++ {
		f.wind[i].draw()
		f.wind[i].update(cellSize, sandPositions, sandParticles)
		if f.wind[i].xPos > float32(rl.GetScreenWidth()) {
			f.wind = append(f.wind[:i], f.wind[i+1:]...)
		}
	}
}

type Wind struct {
	direction int
	xVel      float32
	yVel      float32
	yAcc      float32
	xAcc      float32
	xPos      float32
	yPos      float32
}

func (w *Wind) update(cellSize int, sandPositions [][]int, sandParticles []Grain) {
	if w.direction == 2 {
		w.xVel = float32(rl.GetRandomValue(-50, -70)) / 100
	} else {
		w.xVel = float32(rl.GetRandomValue(50, 70)) / 100
	}
	w.xPos += w.xVel

	// Brownian motion

	var xForce float32 = float32(rl.GetRandomValue(-100, 100))
	xForce = xForce / 3000
	var yForce float32 = float32(rl.GetRandomValue(-100, 100))
	yForce = yForce / 3000

	w.xPos += xForce
	w.yPos += yForce

	// Push Sand
	gridX := (int(w.xPos) / cellSize)
	gridY := (int(w.yPos) / cellSize)
	for i := 0; i < len(sandParticles); i++ {
		if sandParticles[i].boardPos[0] == int32(gridX) && sandParticles[i].boardPos[1] == int32(gridY) {
			sandParticles[i].windPush(rl.NewVector2(w.xVel/2, 0))
		}
	}
}

func (w *Wind) draw() {
	rl.DrawCircle(int32(w.xPos), int32(w.yPos), 1, rl.White)
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

	var selectedTool int = 0 // 0 = Sand, 1 = Fan
	var fans []Fan

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
			// var xForce float32 = float32(rl.GetRandomValue(-100, 100))
			// xForce = xForce / 100000
			// sandParticles[i].addForce(rl.NewVector2(xForce, 0))
			sandPositions = sandParticles[i].update(cellSize, sandPositions, bottomBuffer, sandParticles)
		} // Draw and update all particles

		for i := 0; i < len(fans); i++ {
			fans[i].draw(cellSize)
			fans[i].update(cellSize, sandPositions, sandParticles)
		}

		bottomYPos := int32(rl.GetScreenHeight() - bottomBuffer)
		rightSide := int32(rl.GetScreenWidth())
		rl.DrawLine(0, bottomYPos, rightSide, bottomYPos, rl.White)

		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			if selectedTool == 0 {
				// Check if the mouse is within the grid
				if rl.GetMousePosition().Y < float32(rl.GetScreenHeight()-bottomBuffer) {
					sandParticles = dropSand(sandParticles, cellSize, sandPositions, moistureSelected)
				}
			} else {
				pos := rl.GetMousePosition()
				pos.X = float32(int(pos.X)/int(cellSize)) * float32(cellSize)
				pos.Y = float32(int(pos.Y)/int(cellSize)) * float32(cellSize)
				fans = append(fans, Fan{
					pos:       pos,
					direction: 4,
					strength:  1.0,
					width:     float32(cellSize),
					height:    float32(cellSize * 8),
					spawnRate: 20,
				})
			}
		}

		if rl.IsKeyPressed(rl.KeyC) {
			if selectedTool == 0 {
				selectedTool = 1
			} else {
				selectedTool = 0
			}
		}

		// Slider
		value, xPos := slider(bottomBuffer, buttonX)
		buttonX = xPos
		moistureSelected = value

		drawMouseOutlines(selectedTool, cellSize)

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

func drawMouseOutlines(selectedTool, cellSize int) {
	if selectedTool == 0 {
		drawSandOutline(cellSize)
	} else {
		drawFanOutline(cellSize)
	}
}

func drawFanOutline(cellSize int) {
	mouseX := (rl.GetMouseX() / int32(cellSize)) * int32(cellSize)
	mouseY := (rl.GetMouseY() / int32(cellSize)) * int32(cellSize)
	thickness := float32(1)

	startPos := rl.NewVector2(float32(mouseX), float32(mouseY))
	endPos := rl.NewVector2(float32(mouseX+int32(cellSize)), float32(mouseY))

	rl.DrawLineEx(startPos, endPos, thickness, rl.White)

	yPos := float32(mouseY + (8 * (int32(cellSize))))
	startPos = rl.NewVector2(float32(mouseX), yPos)
	endPos = rl.NewVector2(float32(mouseX+int32(cellSize)), yPos)

	rl.DrawLineEx(startPos, endPos, thickness, rl.White)

	startPos = rl.NewVector2(float32(mouseX+int32(cellSize)), float32(mouseY))
	endPos = rl.NewVector2(float32(mouseX+int32(cellSize)), yPos)

	rl.DrawLineEx(startPos, endPos, thickness, rl.White)

	startPos = rl.NewVector2(float32(mouseX), float32(mouseY))
	endPos = rl.NewVector2(float32(mouseX), yPos)
	rl.DrawLineEx(startPos, endPos, thickness, rl.White)
}

func drawSandOutline(cellSize int) {
	mouseX := (rl.GetMouseX() / int32(cellSize)) * int32(cellSize)
	mouseY := (rl.GetMouseY() / int32(cellSize)) * int32(cellSize)
	thickness := float32(1)

	startPos := rl.NewVector2(float32(mouseX), float32(mouseY))
	endPos := rl.NewVector2(float32(mouseX+int32(cellSize)), float32(mouseY))

	rl.DrawLineEx(startPos, endPos, thickness, rl.White)

	startPos = rl.NewVector2(float32(mouseX), float32(mouseY+int32(cellSize)))
	endPos = rl.NewVector2(float32(mouseX+int32(cellSize)), float32(mouseY+int32(cellSize)))

	rl.DrawLineEx(startPos, endPos, thickness, rl.White)

	startPos = rl.NewVector2(float32(mouseX+int32(cellSize)), float32(mouseY))
	endPos = rl.NewVector2(float32(mouseX+int32(cellSize)), float32(mouseY+int32(cellSize)))

	rl.DrawLineEx(startPos, endPos, thickness, rl.White)

	startPos = rl.NewVector2(float32(mouseX), float32(mouseY))
	endPos = rl.NewVector2(float32(mouseX), float32(mouseY+int32(cellSize)))

	rl.DrawLineEx(startPos, endPos, thickness, rl.White)
}
