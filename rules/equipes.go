package rules

import "sort"

// TeamTally : ce qu'un joueur apporte à son équipe. Points en DEMI-POINTS (deux par victoire et
// par exemption, un par demi-exemption) ; points de match marqués et encaissés, plafonnés à la
// longueur du match comme pour le départage.
type TeamTally struct {
	Team                             string
	Points, PointsFor, PointsAgainst int
}

// TeamStanding : une ligne du classement par équipes ; les ex æquo partagent Rank.
type TeamStanding struct {
	Team      string `json:"team"`
	Rank      int    `json:"rank"`
	Members   int    `json:"members"`
	Points    int    `json:"points"` // demi-points
	Diff      int    `json:"diff"`
	PointsFor int    `json:"points_for"`
}

// RankTeams additionne les apports par équipe et les classe : points, puis différence, puis
// points marqués. Une égalité sur les trois reste un ex æquo ; entre ex æquo, l'ordre des noms
// d'équipe n'est qu'un ordre d'affichage. Un apport sans équipe est ignoré.
func RankTeams(tallies []TeamTally) []TeamStanding {
	idx := map[string]int{}
	var out []TeamStanding
	for _, x := range tallies {
		if x.Team == "" {
			continue
		}
		i, ok := idx[x.Team]
		if !ok {
			i = len(out)
			idx[x.Team] = i
			out = append(out, TeamStanding{Team: x.Team})
		}
		s := &out[i]
		s.Members++
		s.Points += x.Points
		s.Diff += x.PointsFor - x.PointsAgainst
		s.PointsFor += x.PointsFor
	}
	clé := func(s TeamStanding) [3]int { return [3]int{s.Points, s.Diff, s.PointsFor} }
	sort.Slice(out, func(i, j int) bool {
		a, b := clé(out[i]), clé(out[j])
		for k := range a {
			if a[k] != b[k] {
				return a[k] > b[k]
			}
		}
		return out[i].Team < out[j].Team
	})
	for i := range out {
		out[i].Rank = i + 1
		if i > 0 && clé(out[i]) == clé(out[i-1]) {
			out[i].Rank = out[i-1].Rank
		}
	}
	return out
}
