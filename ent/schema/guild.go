package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/disgoorg/snowflake/v2"
	"time"
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
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Guild.
func (Guild) Edges() []ent.Edge {
	return nil
}
