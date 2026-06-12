package main

import (
	"testing"
)

func TestResolveBotRegistryState(t *testing.T) {
	botKey := "1_0xabc"
	idleSet := map[string]struct{}{}
	inGame := map[string]string{}
	tokenInsufficientSet := map[string]struct{}{}

	t.Run("token_insufficient only", func(t *testing.T) {
		tokenInsufficientSet[botKey] = struct{}{}
		got := resolveBotRegistryState(botKey, idleSet, inGame, tokenInsufficientSet)
		if got.State != "token_insufficient" {
			t.Fatalf("state=%q want token_insufficient", got.State)
		}
		if len(got.Warnings) != 0 {
			t.Fatalf("warnings=%v want none", got.Warnings)
		}
		delete(tokenInsufficientSet, botKey)
	})

	t.Run("ingame takes precedence over token_insufficient", func(t *testing.T) {
		inGame[botKey] = "1"
		tokenInsufficientSet[botKey] = struct{}{}
		got := resolveBotRegistryState(botKey, idleSet, inGame, tokenInsufficientSet)
		if got.State != "ingame" {
			t.Fatalf("state=%q want ingame", got.State)
		}
		if len(got.Warnings) != 1 || got.Warnings[0] != "conflict:ingame,token_insufficient" {
			t.Fatalf("warnings=%v want conflict warn", got.Warnings)
		}
		delete(inGame, botKey)
		delete(tokenInsufficientSet, botKey)
	})

	t.Run("idle and token_insufficient conflict", func(t *testing.T) {
		idleSet[botKey] = struct{}{}
		tokenInsufficientSet[botKey] = struct{}{}
		got := resolveBotRegistryState(botKey, idleSet, inGame, tokenInsufficientSet)
		if got.State != "idle" {
			t.Fatalf("state=%q want idle", got.State)
		}
		if len(got.Warnings) != 1 || got.Warnings[0] != "conflict:idle,token_insufficient" {
			t.Fatalf("warnings=%v want conflict warn", got.Warnings)
		}
		delete(idleSet, botKey)
		delete(tokenInsufficientSet, botKey)
	})

	t.Run("orphan when not in any set", func(t *testing.T) {
		got := resolveBotRegistryState(botKey, idleSet, inGame, tokenInsufficientSet)
		if got.State != "orphan" {
			t.Fatalf("state=%q want orphan", got.State)
		}
		if len(got.Warnings) != 0 {
			t.Fatalf("warnings=%v want none", got.Warnings)
		}
	})
}
