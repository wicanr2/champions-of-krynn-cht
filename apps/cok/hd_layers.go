package cok

// HDLayers 是逐幀的牆面、戰鬥與遭遇圖呈現層。主專案規格 010 §6、§7。
// 它只讀原版色號，呼叫端在既有文字與面板之前繪製。
type HDLayers struct {
	combat     *combatHD
	actors     *actorHD
	encounters *encounterHD
	walls      *wallHD
	exact      *exactHD
}

func LoadHDLayers(dir string) (*HDLayers, int, error) {
	combat, err := loadCombatHD(dir)
	if err != nil {
		return nil, 0, err
	}
	actors, err := loadActorHD(dir)
	if err != nil {
		return nil, 0, err
	}
	characters, err := loadCharacterHD(dir)
	if err != nil {
		return nil, 0, err
	}
	actors = mergeCharacterHD(actors, characters)
	effects, err := loadEffectHD(dir)
	if err != nil {
		return nil, 0, err
	}
	actors, err = mergeEffectHD(actors, effects)
	if err != nil {
		return nil, 0, err
	}
	encounters, err := loadEncounterHD(dir)
	if err != nil {
		return nil, 0, err
	}
	walls, err := loadWallHD(dir)
	if err != nil {
		return nil, 0, err
	}
	exact, err := loadExactHD(dir)
	if err != nil {
		return nil, 0, err
	}
	count := 0
	if combat != nil {
		count += len(combat.tiles)
	}
	if actors != nil {
		count += actors.sources
	}
	if encounters != nil {
		count += len(encounters.items)
	}
	if walls != nil {
		count += walls.sources
	}
	if exact != nil {
		count += len(exact.items)
	}
	return &HDLayers{combat: combat, actors: actors, encounters: encounters, walls: walls, exact: exact}, count, nil
}

func (h *HDLayers) Frame(indexed []byte, pal [256][3]uint8) {
	if h == nil {
		return
	}
	h.combat.frame(indexed, pal)
	guard := h.combat != nil && len(h.combat.matches) >= 3
	h.actors.frame(indexed, guard)
	h.encounters.frame(indexed, pal)
	h.walls.frame(indexed, pal)
	h.exact.frame(indexed, pal)
}

func (h *HDLayers) Draw(out, indexed []byte, scale int) {
	if h == nil {
		return
	}
	h.walls.draw(out, indexed, scale)
	h.combat.draw(out, indexed, scale)
	h.actors.draw(out, indexed, scale)
	h.encounters.draw(out, indexed, scale)
	h.exact.draw(out, indexed, scale)
}
