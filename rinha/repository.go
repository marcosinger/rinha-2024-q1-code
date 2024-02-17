package main

import (
	"context"
	"database/sql"
	"errors"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//goland:noinspection SqlNoDataSourceInspection,SqlResolve
func getClient(tx pgx.Tx, clientID int) (Client, error) {
	const query = "SELECT id, nome, limite, saldo FROM clientes WHERE id = $1"
	var c Client
	err := tx.QueryRow(context.Background(), query, clientID).Scan(&c.ID, &c.Name, &c.Limit, &c.Balance)
	if err != nil {
		return Client{}, err
	}
	return c, nil
}

//goland:noinspection SqlNoDataSourceInspection,SqlResolve
func insertTransaction(tx pgx.Tx, t Transaction) error {
	const query = "INSERT INTO transacoes (cliente_id, valor, realizada_em, descricao, tipo) VALUES ($1, $2, now(), $3, $4)"
	_, err := tx.Exec(context.Background(), query, t.ClienteID, t.Value, t.Description, t.Type)
	return err
}

//goland:noinspection SqlNoDataSourceInspection,SqlResolve
func updateSaldo(tx pgx.Tx, clienteID, valor int) error {
	const query = "UPDATE clientes SET saldo = saldo + $1 WHERE id = $2"
	_, err := tx.Exec(context.Background(), query, valor, clienteID)
	return err
}

//goland:noinspection SqlNoDataSourceInspection,SqlResolve
func getClientWithTransactions(dbpool *pgxpool.Pool, clienteID int) (ClientWithTransactions, error) {
	const query = `
    SELECT c.id, c.limite, c.saldo, t.valor, t.tipo, t.descricao, t.realizada_em
    FROM clientes c
    LEFT JOIN transacoes t ON c.id = t.cliente_id
    WHERE c.id = $1 
    ORDER BY t.realizada_em DESC
    LIMIT 10`

	rows, err := dbpool.Query(context.Background(), query, clienteID)
	if err != nil {
		return ClientWithTransactions{}, err
	}
	defer rows.Close()

	var result ClientWithTransactions
	var hasCliente bool
	for rows.Next() {
		var transacao Transaction
		var tipo, desc sql.NullString // Use a nullable type for the Tipo
		var realizadaEm sql.NullTime
		var valor sql.NullInt32

		// Scan the row
		if err := rows.Scan(&result.Client.ID, &result.Client.Limit, &result.Client.Balance, &valor, &tipo, &desc, &realizadaEm); err != nil {
			return ClientWithTransactions{}, err
		}

		var intVal int
		if valor.Valid {
			intVal = int(math.Abs(float64(valor.Int32)))
		}

		if tipo.Valid { // Check if Tipo is not null
			transacao = Transaction{
				ClienteID:   clienteID,
				Description: desc.String,
				Date:        realizadaEm.Time,
				Type:        tipo.String,
				Value:       intVal,
			}
			result.Transacoes = append(result.Transacoes, transacao)
		}

		if !hasCliente {
			hasCliente = true
		}
	}
	if !hasCliente {
		return ClientWithTransactions{}, errors.New("client not found")
	}
	return result, nil
}
