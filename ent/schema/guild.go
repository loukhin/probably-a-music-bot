package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/disgoorg/snowflake/v2"
)

// Guild holds the schema definition for the Guild entity.
type Guild struct {
	ent.Schema
}

// Fields of the Guild.
func (Guild) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("id").Unique().GoType(snowflake.New(time.Now())),
		field.String("name"),
		field.Uint64("player_channel_id").Unique().Optional().Nillable().GoType(snowflake.New(time.Now())),
		field.Uint64("player_message_id").Unique().Optional().Nillable().GoType(snowflake.New(time.Now())),
		field.Time("created_at").Optional().Default(time.Now),
		field.Time("updated_at").Optional().Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Guild.
func (Guild) Edges() []ent.Edge {
	return nil
}

func (Guild) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id").
			Unique(),
	}
}
