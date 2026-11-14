package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Функция создаёт отдельную копию БД для теста
func getTestStore(t *testing.T) ParcelStore {
	t.Helper()

	data, err := os.ReadFile("tracker.db")
	require.NoError(t, err)

	filename := filepath.Join(t.TempDir(), "tracker.db")
	err = os.WriteFile(filename, data, 0600)
	require.NoError(t, err)

	db, err := sql.Open("sqlite", filename)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	err = db.Ping()
	require.NoError(t, err)

	_, err = db.Exec("DELETE FROM parcel")
	require.NoError(t, err)

	return NewParcelStore(db)
}

// Функция возвращает тестовую посылку
func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// Функция проверяет добавление, получение и удаление посылки
func TestAddGetDelete(t *testing.T) {
	store := getTestStore(t)
	parcel := getTestParcel()

	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Positive(t, id)
	parcel.Number = id

	storedParcel, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, parcel, storedParcel)

	err = store.Delete(id)
	require.NoError(t, err)

	_, err = store.Get(id)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// Функция проверяет обновление адреса
func TestSetAddress(t *testing.T) {
	store := getTestStore(t)
	parcel := getTestParcel()

	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Positive(t, id)
	parcel.Number = id

	newAddress := "new test address"
	err = store.SetAddress(id, newAddress)
	require.NoError(t, err)
	parcel.Address = newAddress

	storedParcel, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, parcel, storedParcel)
}

// Функция проверяет обновление статуса
func TestSetStatus(t *testing.T) {
	store := getTestStore(t)
	parcel := getTestParcel()

	id, err := store.Add(parcel)
	require.NoError(t, err)
	require.Positive(t, id)
	parcel.Number = id

	for _, status := range []string{ParcelStatusSent, ParcelStatusDelivered} {
		err = store.SetStatus(id, status)
		require.NoError(t, err)
		parcel.Status = status

		storedParcel, err := store.Get(id)
		require.NoError(t, err)
		require.Equal(t, parcel, storedParcel)
	}
}

// Функция проверяет получение посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	store := getTestStore(t)
	client := 1000
	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	parcelMap := map[int]Parcel{}

	for i := 0; i < len(parcels); i++ {
		parcels[i].Client = client
		id, err := store.Add(parcels[i])
		require.NoError(t, err)
		require.Positive(t, id)

		parcels[i].Number = id
		parcelMap[id] = parcels[i]
	}

	otherParcel := getTestParcel()
	otherParcel.Client = client + 1
	_, err := store.Add(otherParcel)
	require.NoError(t, err)

	storedParcels, err := store.GetByClient(client)
	require.NoError(t, err)
	require.Len(t, storedParcels, len(parcels))

	for _, parcel := range storedParcels {
		require.Contains(t, parcelMap, parcel.Number)
		require.Equal(t, parcelMap[parcel.Number], parcel)
		delete(parcelMap, parcel.Number)
	}
	require.Empty(t, parcelMap)

	storedParcels, err = store.GetByClient(client + 2)
	require.NoError(t, err)
	require.Empty(t, storedParcels)
}

// Функция проверяет запрет изменения адреса и удаления после отправки
func TestParcelRestrictions(t *testing.T) {
	for _, status := range []string{ParcelStatusSent, ParcelStatusDelivered} {
		t.Run(status, func(t *testing.T) {
			store := getTestStore(t)
			parcel := getTestParcel()
			parcel.Status = status

			id, err := store.Add(parcel)
			require.NoError(t, err)
			require.Positive(t, id)
			parcel.Number = id

			err = store.SetAddress(id, "another address")
			require.NoError(t, err)

			storedParcel, err := store.Get(id)
			require.NoError(t, err)
			require.Equal(t, parcel, storedParcel)

			err = store.Delete(id)
			require.NoError(t, err)

			storedParcel, err = store.Get(id)
			require.NoError(t, err)
			require.Equal(t, parcel, storedParcel)
		})
	}
}
