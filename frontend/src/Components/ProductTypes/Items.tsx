import React from "react";

import Endpoint from "../Endpoint";
import ProductTypesContainer from "./ProductTypesContainer";
import {
  transformItemData,
  transformAccountsData,
  itemCategories,
  accountsCategories,
} from "../../dataUtilities";

const Items = (props: any) => (
  <>
    <ProductTypesContainer productType="Item Management">
      <Endpoint
        endpoint="item"
        categories={itemCategories}
        schema="/item/get/"
        description="Retrieve information about an Item, like the institution,
        billed products, available products, and webhook
        information."
        transformData={transformItemData}
        userAuthToken={props.userAuthTokens}
      />
      <Endpoint
        endpoint="accounts"
        schema="/accounts/get"
        categories={accountsCategories}
        description="Retrieve high-level information about all accounts associated with an item."
        transformData={transformAccountsData}
        userAuthToken={props.userAuthToken}
      />
    </ProductTypesContainer>
  </>
);

Items.displayName = "Items";

export default Items;
