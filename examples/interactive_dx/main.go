package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aretw0/lifecycle"
)

func main() {
	reader, _ := lifecycle.OpenTerminal()
	fmt.Println("=== DEMO: Interactive DX (Pure Facade) ===")
	fmt.Println("This demo shows the full 'Foundation for CLIs' power via the lifecycle.* facade.")

	// --- SCENARIO 1: The "I Give Up" (Clean Cancel) ---
	{
		fmt.Println("---------------------------------------------------")
		fmt.Println("[Scenario 1] Basic Cancel")
		fmt.Println("Goal: User sees prompt, decides not to proceed, hits Ctrl+C.")
		fmt.Println("👉 ACTION: Just hit Ctrl+C.")
		fmt.Print("> ")

		ctx := lifecycle.NewSignalContext(context.Background())
		r := lifecycle.NewInterruptibleReader(reader, ctx.Done())

		buf := make([]byte, 1024)
		n, err := r.Read(buf)

		// Facade: lifecycle.IsInterrupted
		if lifecycle.IsInterrupted(err) && n == 0 {
			fmt.Println("\n✅ Operation cancelled cleanly.")
		} else {
			fmt.Printf("\n❌ Unexpected result: n=%d err=%v\n", n, err)
		}

		time.Sleep(500 * time.Millisecond) // Wait for signal to cool down
		ctx.Stop()                         // Cleanup and Unregister Listener
		if ctx.Err() != nil {
			fmt.Println("   (Context was indeed cancelled)")
		}
	}

	time.Sleep(1 * time.Second)
	fmt.Println("\n⚠️  Resetting... (Release Ctrl+C)")
	time.Sleep(2 * time.Second)

	// --- SCENARIO 2: The "Pipeline" (Data Safety) ---
	{
		fmt.Println("---------------------------------------------------")
		fmt.Println("[Scenario 2] Pipeline Safety")
		fmt.Println("Goal: Saves data even if signal arrives simultaneously (Race Condition).")
		fmt.Println("👉 ACTION: Type 'SaveMe' + Enter... I will auto-cancel you.")
		fmt.Print("> ")

		ctx := lifecycle.NewSignalContext(context.Background())
		r := lifecycle.NewInterruptibleReader(reader, ctx.Done())

		// Simulate Race
		go func() {
			time.Sleep(2 * time.Second) // Wait for you to start typing
			fmt.Println("\n[⚡ AUTO-CANCEL ⚡]")
			ctx.Cancel()
		}()

		buf := make([]byte, 1024)
		n, err := r.Read(buf)

		if n == 0 && lifecycle.IsInterrupted(err) {
			fmt.Println("\n🛡️  Aborted during Input Phase (Strict Mode).")
			time.Sleep(500 * time.Millisecond)
			ctx.Stop()
		} else {
			if n > 0 {
				fmt.Printf("\n✅ READ SUCCESS: '%s' (Data Saved - Pipeline Safe)\n", buf[:n])
			} else {
				fmt.Println("\n❌ Failed to save data (Too slow?)")
			}
			time.Sleep(500 * time.Millisecond)
			ctx.Stop() // Cleanup
		}
		if ctx.Err() != nil {
			fmt.Println("   (Context was indeed cancelled)")
		}
	}

	time.Sleep(1 * time.Second)
	fmt.Println("\n⚠️  Resetting... (Release Ctrl+C)")
	time.Sleep(2 * time.Second)

	// --- SCENARIO 3: The "Regret Window" (Two-Stage Safety) ---
	{
		fmt.Println("---------------------------------------------------")
		fmt.Println("[Scenario 3] Two-Stage Safety (Strict Input + Grace Period)")
		fmt.Println("Goal: You submit a command, but change your mind during processing.")
		fmt.Println("STEP 1: Strict Input - Prevents panic typing errors.")
		fmt.Println("STEP 2: Grace Period - Gives you 3s to abort after Enter.")
		fmt.Println("👉 ACTION: Type 'Delete', hit Enter, THEN hit Ctrl+C during the countdown.")
		fmt.Print("> ")

		ctx := lifecycle.NewSignalContext(context.Background())
		r := lifecycle.NewInterruptibleReader(reader, ctx.Done())

		// 1. Strict Input Phase
		buf := make([]byte, 1024)
		n, err := r.ReadInteractive(buf)

		if n == 0 && lifecycle.IsInterrupted(err) {
			fmt.Println("\n🛡️  Aborted during Input Phase (Strict Mode).")
			time.Sleep(500 * time.Millisecond)
			ctx.Stop()
		} else {

			// 2. Grace Period Phase
			fmt.Printf("\n✅ Input Accepted: '%s'", buf[:n])
			fmt.Println("⏳ Processing in 3 seconds... (Hit Ctrl+C to Abort!)")

			if err := lifecycle.Sleep(ctx, 3*time.Second); err != nil {
				fmt.Println("\n🛑 ABORTED during Grace Period (User Regret).")
				fmt.Printf("   Reason: %v\n", err)
			} else {
				fmt.Println("\n🚀 COMMAND EXECUTED (No regret).")
			}

			time.Sleep(500 * time.Millisecond)
			ctx.Stop() // Cleanup
		}

		if ctx.Err() != nil {
			fmt.Println("   (Context was indeed cancelled)")
		}
	}

	fmt.Println("\n=== Demo Complete ===")
}
