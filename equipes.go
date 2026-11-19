package tournoi

import "github.com/kevung/nicomaque/rules"

// TeamRank : une ligne du classement par équipes (rules.RankTeams). Points en demi-points.
type TeamRank = rules.TeamStanding

// TeamRanking : le classement par équipes, tous les matchs terminés et toutes les phases
// confondus. Chaque membre (Player.Team), retiré ou non, apporte ses victoires et exemptions de
// toutes les phases et ses points de match, plafonnés à la longueur du match ; un forfait vaut
// la longueur au vainqueur, zéro au perdant. Le classement individuel n'en dépend pas.
func (s *State) TeamRanking() []TeamRank {
	ids := make([]PlayerID, 0, len(s.Players))
	for id, pl := range s.Players {
		if pl != nil && pl.Team != "" {
			ids = append(ids, id)
		}
	}
	ids = sortedIDs(ids)
	apport := make(map[PlayerID]*rules.TeamTally, len(ids))
	tallies := make([]rules.TeamTally, len(ids))
	for i, id := range ids {
		tallies[i].Team = s.Players[id].Team
		apport[id] = &tallies[i]
	}
	for _, ph := range s.Phases {
		for id, t := range apport {
			t.Points += 2 * (ph.Wins[id] + ph.Byes[id])
			for _, r := range ph.HalfByes[id] {
				if r <= ph.Round {
					t.Points++
				}
			}
		}
	}
	for _, mid := range s.MatchOrder {
		m := s.Matches[mid]
		if m.Status != Finished {
			continue
		}
		sa, sb := min(m.ScoreA, m.Length), min(m.ScoreB, m.Length)
		if m.Forfeit {
			sa, sb = 0, 0
			if m.Winner == m.A {
				sa = m.Length
			} else {
				sb = m.Length
			}
		}
		if t := apport[m.A]; t != nil {
			t.PointsFor, t.PointsAgainst = t.PointsFor+sa, t.PointsAgainst+sb
		}
		if t := apport[m.B]; t != nil {
			t.PointsFor, t.PointsAgainst = t.PointsFor+sb, t.PointsAgainst+sa
		}
	}
	return rules.RankTeams(tallies)
}

// TeamMeeting : la « rencontre » entre les équipes a et b, regroupée après coup — les matchs
// terminés entre un membre de a et un membre de b, dans l'ordre du journal. Les matchs restent
// individuels : il n'y a pas d'échiquiers.
func (s *State) TeamMeeting(a, b string) []MatchID {
	équipe := func(p PlayerID) string {
		if pl := s.Players[p]; pl != nil {
			return pl.Team
		}
		return ""
	}
	var out []MatchID
	for _, mid := range s.MatchOrder {
		m := s.Matches[mid]
		if m.Status != Finished || a == "" || b == "" {
			continue
		}
		ta, tb := équipe(m.A), équipe(m.B)
		if ta == a && tb == b || ta == b && tb == a {
			out = append(out, mid)
		}
	}
	return out
}
