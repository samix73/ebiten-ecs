package ecs

import (
	"reflect"
	"slices"
	"testing"

	"github.com/samix73/game/helpers"
	"github.com/stretchr/testify/require"
)

func TestNewArchetype(t *testing.T) {
	tests := []struct {
		name          string
		signatureMask Bitmask
		want          *Archetype
	}{
		{
			name:          "valid: no components",
			signatureMask: NewBitmask(1),
			want: &Archetype{
				signature:    NewBitmask(1),
				entities:     []EntityID{},
				components:   map[ComponentID]reflect.Value{},
				entityLookup: map[EntityID]int{},
			},
		},
		{
			name:          "valid: one component",
			signatureMask: NewBitmask(1),
			want: &Archetype{
				signature:    NewBitmask(1),
				entities:     []EntityID{},
				components:   map[ComponentID]reflect.Value{},
				entityLookup: map[EntityID]int{},
			},
		},
		{
			name:          "valid: two components",
			signatureMask: NewBitmask(1, 2),
			want: &Archetype{
				signature:    NewBitmask(1, 2),
				entities:     []EntityID{},
				components:   map[ComponentID]reflect.Value{},
				entityLookup: map[EntityID]int{},
			},
		},
		{
			name:          "valid: no bit set",
			signatureMask: NewBitmask(1),
			want: &Archetype{
				signature:    NewBitmask(1),
				entities:     []EntityID{},
				components:   map[ComponentID]reflect.Value{},
				entityLookup: map[EntityID]int{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewArchetype(tt.signatureMask)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestArchetype_AddEntity(t *testing.T) {
	err := RegisterComponent[int]()
	require.NoError(t, err)

	intComponentID, ok := getComponentID(reflect.TypeFor[int]())
	require.True(t, ok)

	err = RegisterComponent[float64]()
	require.NoError(t, err)

	float64ComponentID, ok := getComponentID(reflect.TypeFor[float64]())
	require.True(t, ok)

	type entityToAdd struct {
		entityID   EntityID
		components map[ComponentID]any
		wantErr    bool
	}

	tests := []struct {
		name          string
		signatureMask Bitmask
		entitiesToAdd []entityToAdd
	}{
		{
			name:          "valid: no components",
			signatureMask: NewBitmask(),
			entitiesToAdd: []entityToAdd{
				{
					entityID:   1,
					components: map[ComponentID]any{},
					wantErr:    false,
				},
			},
		},
		{
			name:          "valid: one component",
			signatureMask: NewBitmask(intComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID:   1,
					components: map[ComponentID]any{intComponentID: helpers.New(1)},
					wantErr:    false,
				},
			},
		},
		{
			name:          "valid: two components",
			signatureMask: NewBitmask(intComponentID, float64ComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID: 1,
					components: map[ComponentID]any{
						intComponentID:     helpers.New(1),
						float64ComponentID: helpers.New(2.0),
					},
					wantErr: false,
				},
			},
		},
		{
			name:          "invalid: nil component",
			signatureMask: NewBitmask(intComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID: 1,
					components: map[ComponentID]any{
						intComponentID: nil,
					},
					wantErr: true,
				},
			},
		},
		{
			name:          "invalid: component type not registered",
			signatureMask: NewBitmask(intComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID: 1,
					components: map[ComponentID]any{
						2323: helpers.New("test"),
					},
					wantErr: true,
				},
			},
		},
		{
			name:          "invalid: same entity ID",
			signatureMask: NewBitmask(intComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID:   1,
					components: map[ComponentID]any{intComponentID: helpers.New(1)},
					wantErr:    false,
				},
				{
					entityID:   1,
					components: map[ComponentID]any{intComponentID: helpers.New(2)},
					wantErr:    true,
				},
			},
		},
		{
			name:          "invalid: component is not pointer",
			signatureMask: NewBitmask(intComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID: 1,
					components: map[ComponentID]any{
						intComponentID: 1,
					},
					wantErr: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := NewArchetype(tt.signatureMask)

			for _, entityToAdd := range tt.entitiesToAdd {
				gotErr := a.AddEntity(entityToAdd.entityID, entityToAdd.components)
				if entityToAdd.wantErr {
					require.Error(t, gotErr)
					return
				}
				require.NoError(t, gotErr)

				require.True(t, slices.ContainsFunc(a.entities, func(e EntityID) bool {
					return e == entityToAdd.entityID
				}))

				require.Contains(t, a.entityLookup, entityToAdd.entityID)

				for componentID, componentData := range entityToAdd.components {
					gotComponent, exists := a.GetComponent(entityToAdd.entityID, componentID)
					require.Truef(t, exists, "component %v not found", componentID)

					require.Equal(t, componentData, gotComponent)
				}
			}
		})
	}
}

func TestArchetype_RemoveEntity(t *testing.T) {
	err := RegisterComponent[int]()
	require.NoError(t, err)

	intComponentID, ok := getComponentID(reflect.TypeFor[int]())
	require.True(t, ok)

	err = RegisterComponent[float64]()
	require.NoError(t, err)

	float64ComponentID, ok := getComponentID(reflect.TypeFor[float64]())
	require.True(t, ok)

	type entityToAdd struct {
		entityID       EntityID
		componentsData map[ComponentID]any
	}

	type entityToRemove struct {
		entityID EntityID
		want     map[ComponentID]any
		wantErr  bool
	}

	tests := []struct {
		name             string
		entitiesToAdd    []entityToAdd
		entitiesToRemove []entityToRemove
		signatureMask    Bitmask
	}{
		{
			name:          "valid: no components",
			signatureMask: NewBitmask(),
			entitiesToAdd: []entityToAdd{
				{
					entityID:       0,
					componentsData: map[ComponentID]any{},
				},
			},
			entitiesToRemove: []entityToRemove{
				{
					entityID: 0,
					want:     map[ComponentID]any{},
					wantErr:  false,
				},
			},
		},
		{
			name:          "valid: one component",
			signatureMask: NewBitmask(intComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID: 0,
					componentsData: map[ComponentID]any{
						intComponentID: helpers.New(1),
					},
				},
			},
			entitiesToRemove: []entityToRemove{
				{
					entityID: 0,
					want: map[ComponentID]any{
						intComponentID: helpers.New(1),
					},
					wantErr: false,
				},
			},
		},
		{
			name:          "valid: two components",
			signatureMask: NewBitmask(intComponentID, float64ComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID: 0,
					componentsData: map[ComponentID]any{
						intComponentID:     helpers.New(1),
						float64ComponentID: helpers.New(2.0),
					},
				},
			},
			entitiesToRemove: []entityToRemove{
				{
					entityID: 0,
					want: map[ComponentID]any{
						intComponentID:     helpers.New(1),
						float64ComponentID: helpers.New(2.0),
					},
					wantErr: false,
				},
			},
		},
		{
			name:          "invalid: entity not found",
			signatureMask: NewBitmask(intComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID: 0,
					componentsData: map[ComponentID]any{
						intComponentID: helpers.New(1),
					},
				},
			},
			entitiesToRemove: []entityToRemove{
				{
					entityID: 1,
					want:     nil,
					wantErr:  true,
				},
			},
		},
		{
			name:          "invalid: remove same entity twice",
			signatureMask: NewBitmask(intComponentID),
			entitiesToAdd: []entityToAdd{
				{
					entityID: 0,
					componentsData: map[ComponentID]any{
						intComponentID: helpers.New(1),
					},
				},
			},
			entitiesToRemove: []entityToRemove{
				{
					entityID: 0,
					want: map[ComponentID]any{
						intComponentID: helpers.New(1),
					},
					wantErr: false,
				},
				{
					entityID: 0,
					want: map[ComponentID]any{
						intComponentID: helpers.New(1),
					},
					wantErr: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := NewArchetype(tt.signatureMask)

			for _, entityToAdd := range tt.entitiesToAdd {
				gotErr := a.AddEntity(entityToAdd.entityID, entityToAdd.componentsData)
				require.NoError(t, gotErr)
			}

			for _, entityToRemove := range tt.entitiesToRemove {
				got, gotErr := a.RemoveEntity(entityToRemove.entityID)
				if entityToRemove.wantErr {
					require.Error(t, gotErr)
					return
				}
				require.NoError(t, gotErr)
				require.Equal(t, entityToRemove.want, got)
				require.False(t, slices.ContainsFunc(a.entities, func(e EntityID) bool {
					return e == entityToRemove.entityID
				}))
				for componentType := range entityToRemove.want {
					exists := a.HasComponent(entityToRemove.entityID, componentType)
					require.False(t, exists)
				}
			}
		})
	}
}

func TestArchetype_MatchesQuery(t *testing.T) {
	t.Parallel()

	// Archetype has components 1 and 2
	archMask := NewBitmask()
	archMask.Set(1)
	archMask.Set(2)
	arch := NewArchetype(archMask)

	// Query for 1 and 2 -> Match
	query1 := NewBitmask()
	query1.Set(1)
	query1.Set(2)
	require.True(t, arch.MatchesQuery(query1))

	// Query for only 1 -> Match (Archetype has all of query)
	query2 := NewBitmask()
	query2.Set(1)
	require.True(t, arch.MatchesQuery(query2))

	// Query for 1, 2, and 3 -> No Match (Archetype missing 3)
	query3 := NewBitmask()
	query3.Set(1)
	query3.Set(2)
	query3.Set(3)
	require.False(t, arch.MatchesQuery(query3))
}

func TestArchetype_SignatureMatches(t *testing.T) {
	t.Parallel()

	// Archetype has components 1 and 2
	archMask := NewBitmask()
	archMask.Set(1)
	archMask.Set(2)
	arch := NewArchetype(archMask)

	// Exact match -> Match
	sig1 := NewBitmask()
	sig1.Set(1)
	sig1.Set(2)
	require.True(t, arch.SignatureMatches(sig1))

	// Subset -> No Match (Exact match required)
	sig2 := NewBitmask()
	sig2.Set(1)
	require.False(t, arch.SignatureMatches(sig2))

	// Superset -> No Match
	sig3 := NewBitmask()
	sig3.Set(1)
	sig3.Set(2)
	sig3.Set(3)
	require.False(t, arch.SignatureMatches(sig3))
}
