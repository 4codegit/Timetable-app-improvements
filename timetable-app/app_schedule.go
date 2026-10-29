package main

import (
	"timetable/internal/domain"
)

// ---- Scheduling ----

func (a *App) ListSchedule(schoolID int) ([]domain.ScheduleEntry, error) {
	return a.store.ListSchedule(schoolID)
}

func (a *App) MoveEntry(schoolID, id, day, slot int) error {
	return a.store.MoveEntry(schoolID, id, day, slot)
}

func (a *App) MoveEntries(schoolID int, ids []int, day, slot int) error {
	return a.store.MoveEntries(schoolID, ids, day, slot)
}

func (a *App) SwapEntries(schoolID, id1, day1, slot1, id2, day2, slot2 int) error {
	return a.store.SwapEntries(schoolID, id1, day1, slot1, id2, day2, slot2)
}

func (a *App) ReplaceSchedule(schoolID int, entries []domain.ScheduleEntry) error {
	return a.store.ReplaceSchedule(schoolID, entries)
}

func (a *App) DeleteScheduleEntry(id int) error {
	return a.store.DeleteScheduleEntry(id)
}
