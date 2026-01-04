package ui

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
)

type SpawnProgress struct {
	spinner   *pterm.SpinnerPrinter
	multi     *pterm.MultiPrinter
	bars      map[string]*pterm.ProgressbarPrinter
	boxStates map[string]string
	total     int
	ready     int
}

func NewSpawnProgress(total int) *SpawnProgress {
	return &SpawnProgress{
		boxStates: make(map[string]string),
		bars:      make(map[string]*pterm.ProgressbarPrinter),
		total:     total,
	}
}

func (sp *SpawnProgress) Start() {
	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("Fleet Provisioning")
	fmt.Println()
}

func (sp *SpawnProgress) StartSpawning() {
	sp.spinner, _ = pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start(fmt.Sprintf("Spawning %d instances...", sp.total))
}

func (sp *SpawnProgress) BoxSpawned(name string) {
	sp.boxStates[name] = "spawned"
	if sp.spinner != nil {
		sp.spinner.UpdateText(fmt.Sprintf("Spawned %d/%d instances...", len(sp.boxStates), sp.total))
	}
}

func (sp *SpawnProgress) SpawningDone() {
	if sp.spinner != nil {
		sp.spinner.Success(fmt.Sprintf("All %d instances spawned", sp.total))
	}
}

func (sp *SpawnProgress) StartWaiting() {
	sp.spinner, _ = pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start("Waiting for instances to become ready...")
}

func (sp *SpawnProgress) UpdateBoxStatus(name, status, ip string) {
	sp.boxStates[name] = status
	ready := 0
	for _, s := range sp.boxStates {
		if s == "running" || s == "active" {
			ready++
		}
	}
	sp.ready = ready

	if sp.spinner != nil {
		sp.spinner.UpdateText(fmt.Sprintf("Waiting for instances... %d/%d ready", ready, sp.total))
	}
}

func (sp *SpawnProgress) WaitingDone() {
	if sp.spinner != nil {
		sp.spinner.Success(fmt.Sprintf("All %d instances ready", sp.total))
	}
}

func (sp *SpawnProgress) Done() {
	pterm.Success.Println("Fleet provisioning complete")
	fmt.Println()
}

type BuildProgress struct {
	fleetSize int
	boxes     map[string]*boxProgress
}

type boxProgress struct {
	name     string
	spinner  *pterm.SpinnerPrinter
	step     string
	stepNum  int
	total    int
	finished bool
	success  bool
}

func NewBuildProgress(fleetSize int) *BuildProgress {
	return &BuildProgress{
		fleetSize: fleetSize,
		boxes:     make(map[string]*boxProgress),
	}
}

func (bp *BuildProgress) Start(recipeName string) {
	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgWhite)).
		Printf("Building with recipe: %s", recipeName)
	fmt.Println()
}

func (bp *BuildProgress) StartBox(name string, totalSteps int) {
	spinner, _ := pterm.DefaultSpinner.
		WithRemoveWhenDone(false).
		Start(fmt.Sprintf("[%s] Starting build...", name))

	bp.boxes[name] = &boxProgress{
		name:    name,
		spinner: spinner,
		total:   totalSteps,
	}
}

func (bp *BuildProgress) UpdateStep(boxName, stepName string, stepNum int) {
	if box, ok := bp.boxes[boxName]; ok {
		box.step = stepName
		box.stepNum = stepNum
		if box.spinner != nil {
			box.spinner.UpdateText(fmt.Sprintf("[%s] Step %d/%d: %s", boxName, stepNum, box.total, stepName))
		}
	}
}

func (bp *BuildProgress) BoxSuccess(boxName string) {
	if box, ok := bp.boxes[boxName]; ok {
		box.finished = true
		box.success = true
		if box.spinner != nil {
			box.spinner.Success(fmt.Sprintf("[%s] Build complete", boxName))
		}
	}
}

func (bp *BuildProgress) BoxFailed(boxName string, err string) {
	if box, ok := bp.boxes[boxName]; ok {
		box.finished = true
		box.success = false
		if box.spinner != nil {
			box.spinner.Fail(fmt.Sprintf("[%s] Build failed: %s", boxName, err))
		}
	}
}

func (bp *BuildProgress) Done() {
	success := 0
	for _, box := range bp.boxes {
		if box.success {
			success++
		}
	}

	fmt.Println()
	if success == bp.fleetSize {
		pterm.Success.Printfln("Build complete: %d/%d successful", success, bp.fleetSize)
	} else {
		pterm.Warning.Printfln("Build complete: %d/%d successful", success, bp.fleetSize)
	}
}

func PrintFleetTable(boxes []FleetBox) {
	tableData := pterm.TableData{
		{"Name", "Status", "IP", "Duration"},
	}

	for _, box := range boxes {
		status := box.Status
		switch status {
		case "running", "active":
			status = pterm.Green(status)
		case "provisioning", "booting":
			status = pterm.Yellow(status)
		default:
			status = pterm.Gray(status)
		}

		tableData = append(tableData, []string{
			box.Name,
			status,
			box.IP,
			box.Duration,
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

type FleetBox struct {
	Name     string
	Status   string
	IP       string
	Duration string
}

func ShowBuildSummary(results []BuildResult) {
	fmt.Println()
	pterm.DefaultSection.Println("Build Summary")

	tableData := pterm.TableData{
		{"Box", "Status", "Duration", "Steps"},
	}

	for _, r := range results {
		status := "Success"
		if !r.Success {
			status = pterm.Red("Failed")
		} else {
			status = pterm.Green("Success")
		}

		tableData = append(tableData, []string{
			r.BoxName,
			status,
			r.Duration.Round(time.Second).String(),
			fmt.Sprintf("%d", r.StepsCompleted),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

type BuildResult struct {
	BoxName        string
	Success        bool
	Duration       time.Duration
	StepsCompleted int
	Error          string
}

func Info(msg string) {
	pterm.Info.Println(msg)
}

func Success(msg string) {
	pterm.Success.Println(msg)
}

func Warning(msg string) {
	pterm.Warning.Println(msg)
}

func Error(msg string) {
	pterm.Error.Println(msg)
}

func Fatal(msg string) {
	pterm.Fatal.Println(msg)
}

type WorkflowProgress struct {
	fleetSize int
	boxes     map[string]*workflowBoxProgress
	spinner   *pterm.SpinnerPrinter
}

type workflowBoxProgress struct {
	name       string
	spinner    *pterm.SpinnerPrinter
	step       string
	stepNum    int
	totalSteps int
	finished   bool
	success    bool
}

func NewWorkflowProgress(fleetSize int) *WorkflowProgress {
	return &WorkflowProgress{
		fleetSize: fleetSize,
		boxes:     make(map[string]*workflowBoxProgress),
	}
}

func (wp *WorkflowProgress) Start(workflowName string, totalSteps int) {
	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgMagenta)).
		WithTextStyle(pterm.NewStyle(pterm.FgWhite)).
		Printf("Running workflow: %s (%d steps)", workflowName, totalSteps)
	fmt.Println()
}

func (wp *WorkflowProgress) StartSetup() {
	wp.spinner, _ = pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start("Running setup commands...")
}

func (wp *WorkflowProgress) SetupDone() {
	if wp.spinner != nil {
		wp.spinner.Success("Setup complete")
	}
}

func (wp *WorkflowProgress) StartChunking(inputFile string) {
	wp.spinner, _ = pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start(fmt.Sprintf("Splitting input file: %s", inputFile))
}

func (wp *WorkflowProgress) ChunkingDone(chunks int) {
	if wp.spinner != nil {
		wp.spinner.Success(fmt.Sprintf("Input split into %d chunks", chunks))
	}
}

func (wp *WorkflowProgress) StartBox(name string, totalSteps int) {
	spinner, _ := pterm.DefaultSpinner.
		WithRemoveWhenDone(false).
		Start(fmt.Sprintf("[%s] Starting workflow...", name))

	wp.boxes[name] = &workflowBoxProgress{
		name:       name,
		spinner:    spinner,
		totalSteps: totalSteps,
	}
}

func (wp *WorkflowProgress) UpdateStep(boxName, stepName string, stepNum int) {
	if box, ok := wp.boxes[boxName]; ok {
		box.step = stepName
		box.stepNum = stepNum
		if box.spinner != nil {
			box.spinner.UpdateText(fmt.Sprintf("[%s] Step %d/%d: %s", boxName, stepNum, box.totalSteps, stepName))
		}
	}
}

func (wp *WorkflowProgress) BoxSuccess(boxName string) {
	if box, ok := wp.boxes[boxName]; ok {
		box.finished = true
		box.success = true
		if box.spinner != nil {
			box.spinner.Success(fmt.Sprintf("[%s] Workflow complete", boxName))
		}
	}
}

func (wp *WorkflowProgress) BoxFailed(boxName string, err string) {
	if box, ok := wp.boxes[boxName]; ok {
		box.finished = true
		box.success = false
		if box.spinner != nil {
			box.spinner.Fail(fmt.Sprintf("[%s] Workflow failed: %s", boxName, err))
		}
	}
}

func (wp *WorkflowProgress) StartAggregating() {
	wp.spinner, _ = pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start("Aggregating results...")
}

func (wp *WorkflowProgress) AggregatingDone(outputFile string) {
	if wp.spinner != nil {
		wp.spinner.Success(fmt.Sprintf("Results saved to: %s", outputFile))
	}
}

func (wp *WorkflowProgress) Done() {
	success := 0
	for _, box := range wp.boxes {
		if box.success {
			success++
		}
	}

	fmt.Println()
	if success == wp.fleetSize {
		pterm.Success.Printfln("Workflow complete: %d/%d successful", success, wp.fleetSize)
	} else {
		pterm.Warning.Printfln("Workflow complete: %d/%d successful", success, wp.fleetSize)
	}
}
