//go:build ignore

// Stress test runner with built-in profiling support
// This program runs the atomic collector stress test with memory and goroutine monitoring
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"time"
)

func main() {
	cpuProfile := flag.String("cpuprofile", "", "write cpu profile to file")
	memProfile := flag.String("memprofile", "", "write memory profile to file")
	goroutineProfile := flag.String("goroutineprofile", "", "write goroutine profile to file")
	monitorInterval := flag.Duration("monitor-interval", 10*time.Second, "interval for monitoring stats")
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("Atomic Collector Stress Test with Profiling")
	fmt.Println("========================================")
	fmt.Println()

	// Enable CPU profiling if requested
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("✓ CPU profiling enabled: %s\n", *cpuProfile)
	}

	// Start monitoring goroutine
	stopMonitor := make(chan struct{})
	monitorDone := make(chan struct{})

	go func() {
		defer close(monitorDone)
		ticker := time.NewTicker(*monitorInterval)
		defer ticker.Stop()

		fmt.Println("\nStarting resource monitoring...")
		fmt.Println("Time\t\tGoroutines\tMemAlloc(MB)\tSys(MB)\t\tNumGC")
		fmt.Println("----\t\t----------\t------------\t-------\t\t-----")

		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				fmt.Printf("%s\t%d\t\t%.2f\t\t%.2f\t\t%d\n",
					time.Now().Format("15:04:05"),
					runtime.NumGoroutine(),
					float64(m.Alloc)/1024/1024,
					float64(m.Sys)/1024/1024,
					m.NumGC,
				)
			case <-stopMonitor:
				return
			}
		}
	}()

	// Set environment variable for atomic collector
	os.Setenv("LUNGE_USE_ATOMIC_COLLECTOR", "true")
	fmt.Println("✓ Atomic collector enabled")
	fmt.Println()

	// Record initial stats
	var initialStats runtime.MemStats
	runtime.ReadMemStats(&initialStats)
	initialGoroutines := runtime.NumGoroutine()

	fmt.Printf("Initial state:\n")
	fmt.Printf("  Goroutines: %d\n", initialGoroutines)
	fmt.Printf("  Memory Allocated: %.2f MB\n", float64(initialStats.Alloc)/1024/1024)
	fmt.Printf("  System Memory: %.2f MB\n", float64(initialStats.Sys)/1024/1024)
	fmt.Println()

	// Run the stress test
	fmt.Println("Starting stress test...")
	fmt.Println()

	cmd := exec.Command(".\\lunge.exe", "perf", "examples/atomic-collector-stress-test.json")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "LUNGE_USE_ATOMIC_COLLECTOR=true")

	startTime := time.Now()
	err := cmd.Run()
	elapsed := time.Since(startTime)

	// Stop monitoring
	close(stopMonitor)
	<-monitorDone

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("Test Completed")
	fmt.Println("========================================")
	fmt.Printf("Duration: %s\n", elapsed)
	fmt.Println()

	// Record final stats
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)
	finalGoroutines := runtime.NumGoroutine()

	fmt.Printf("Final state:\n")
	fmt.Printf("  Goroutines: %d (delta: %+d)\n", finalGoroutines, finalGoroutines-initialGoroutines)
	fmt.Printf("  Memory Allocated: %.2f MB (delta: %+.2f MB)\n",
		float64(finalStats.Alloc)/1024/1024,
		float64(finalStats.Alloc-initialStats.Alloc)/1024/1024)
	fmt.Printf("  System Memory: %.2f MB (delta: %+.2f MB)\n",
		float64(finalStats.Sys)/1024/1024,
		float64(finalStats.Sys-initialStats.Sys)/1024/1024)
	fmt.Printf("  Total GC Runs: %d\n", finalStats.NumGC-initialStats.NumGC)
	fmt.Println()

	// Check for goroutine leaks
	if finalGoroutines > initialGoroutines+5 {
		fmt.Printf("⚠ WARNING: Possible goroutine leak detected! (+%d goroutines)\n", finalGoroutines-initialGoroutines)
	} else {
		fmt.Println("✓ No goroutine leaks detected")
	}

	// Write memory profile if requested
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		fmt.Printf("✓ Memory profile written to: %s\n", *memProfile)
	}

	// Write goroutine profile if requested
	if *goroutineProfile != "" {
		f, err := os.Create(*goroutineProfile)
		if err != nil {
			log.Fatal("could not create goroutine profile: ", err)
		}
		defer f.Close()
		if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
			log.Fatal("could not write goroutine profile: ", err)
		}
		fmt.Printf("✓ Goroutine profile written to: %s\n", *goroutineProfile)
	}

	fmt.Println()

	if err != nil {
		fmt.Printf("✗ Test failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Test completed successfully!")
	fmt.Println()
	fmt.Println("To analyze profiles:")
	if *cpuProfile != "" {
		fmt.Printf("  go tool pprof %s\n", *cpuProfile)
	}
	if *memProfile != "" {
		fmt.Printf("  go tool pprof %s\n", *memProfile)
	}
	if *goroutineProfile != "" {
		fmt.Printf("  go tool pprof %s\n", *goroutineProfile)
	}
}
