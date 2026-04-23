package adapter

type HybridPetRepository struct {
	mysql *MySQLPetAdapter
	mongo *MongoPetAdapter
}

func NewHybridPetRepository(
	mysql *MySQLPetAdapter,
	mongo *MongoPetAdapter,
) *HybridPetRepository {
	return &HybridPetRepository{
		mysql: mysql,
		mongo: mongo,
	}
}